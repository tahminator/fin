import { GitHubClient, Utils } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import {
  DOCKER_IMAGE,
  IMAGE_WAIT_ATTEMPTS,
  IMAGE_WAIT_INTERVAL_MS,
  stagingTag,
} from "../lib/consts";
import { imageTagExists } from "../lib/docker-hub";
import { parseCiEnv } from "../lib/env";
import { describeSchemaVersion, migrateUp, readSchemaVersion } from "../lib/migrations";
import {
  deployToProduction,
  readProductionTag,
  readTryState,
  reportToPr,
  type TryState,
} from "../lib/production";
import { completeStatusCheck, startStatusCheck, updateStatusCheck } from "../lib/status-check";

const CHECK_NAME = "Try in production";

const { prId, sha, author, runUrl } = await yargs(hideBin(process.argv))
  .option("prId", {
    type: "number",
    describe: "Pull request whose staging image is deployed",
    demandOption: true,
  })
  .option("sha", {
    type: "string",
    describe: "Full SHA of the pull request head; its staging image must already exist",
    demandOption: true,
  })
  .option("author", {
    type: "string",
    describe: "Username of the person who triggered the command",
    demandOption: true,
  })
  .option("runUrl", {
    type: "string",
    describe: "Workflow run URL linked from the PR comment",
    demandOption: true,
  })
  .strict()
  .parse();

async function main() {
  const { _GITHUB_APP_APP_ID, _GITHUB_APP_INSTALLATION_ID, _GITHUB_APP_PEM_CONTENT, DATABASE_URL } =
    parseCiEnv([
      "_GITHUB_APP_APP_ID",
      "_GITHUB_APP_INSTALLATION_ID",
      "_GITHUB_APP_PEM_CONTENT",
      "DATABASE_URL",
    ]);

  const ghClient = await GitHubClient.createWithGithubAppToken({
    appId: _GITHUB_APP_APP_ID,
    installationId: _GITHUB_APP_INSTALLATION_ID,
    privateKey: _GITHUB_APP_PEM_CONTENT,
  });

  const tag = stagingTag(sha);
  const image = `${DOCKER_IMAGE}:${tag}`;
  let state: TryState | undefined;
  let checkRunId: number | undefined;
  let migration: string | undefined;

  try {
    checkRunId = await startStatusCheck(ghClient, {
      name: CHECK_NAME,
      sha,
      detailsUrl: runUrl,
      output: {
        title: "Waiting for the staging image",
        summary: `Waiting for CI to push \`${image}\` before deploying it to production.`,
      },
    });

    state = await readTryState(ghClient, prId);

    const pushed = await Utils.waitUntil({
      attempts: IMAGE_WAIT_ATTEMPTS,
      intervalMs: IMAGE_WAIT_INTERVAL_MS,
      predicate: async (attempt) => {
        try {
          if (await imageTagExists(tag)) {
            return true;
          }
          console.log(`Waiting for ${image} (${attempt}/${IMAGE_WAIT_ATTEMPTS})`);
        } catch (e) {
          console.warn(e);
        }
        return false;
      },
    });
    if (!pushed) {
      throw new Error(
        `${image} was never pushed; check CI's "Build staging image" job on ${sha}, then comment /try again.`,
      );
    }

    const current = await readProductionTag();
    // re-trying while this PR's earlier /try is still live keeps the original image to revert to
    const revertTo = state?.tried === current ? state.revertTo : current;
    if (revertTo === tag) {
      throw new Error(
        `Production already runs \`${tag}\`, but this PR has no recorded image to revert to.`,
      );
    }

    const before = await readSchemaVersion(DATABASE_URL);
    if (before.dirty) {
      throw new Error(
        `The production database is dirty at ${describeSchemaVersion(before)}, so nothing was migrated or deployed. Fix it manually, then comment /try again.`,
      );
    }

    await updateStatusCheck(ghClient, checkRunId, {
      title: "Migrating production database",
      summary: `Applying pending migrations from this PR to the production database, currently at ${describeSchemaVersion(before)}.`,
    });

    await migrateUp(DATABASE_URL);

    const after = await readSchemaVersion(DATABASE_URL);
    migration =
      before.version === after.version
        ? `No pending migrations (${describeSchemaVersion(after)}).`
        : `Migrated production from ${describeSchemaVersion(before)} to ${describeSchemaVersion(after)}.`;

    await updateStatusCheck(ghClient, checkRunId, {
      title: "Deploying to production",
      summary: `${migration} Deploying \`${image}\` to production, replacing \`${current}\`.`,
    });

    if (current !== tag) {
      await deployToProduction(ghClient, tag);
    }
    state = { tried: tag, revertTo };

    const summary = `@${author} deployed \`${image}\` (${sha}) to production with \`/try\`, replacing \`${current}\`. ([run](${runUrl}))

${migration} Comment \`/revert\` to restore \`${revertTo}\`; it does not roll back database migrations, so the restored image runs against the migrated schema.`;

    await reportToPr(ghClient, prId, `### Production is running this PR\n\n${summary}`, state);
    await completeStatusCheck(ghClient, checkRunId, "success", {
      title: "Production is running this PR",
      summary,
    });
  } catch (error) {
    const summary = `${error instanceof Error ? error.message : String(error)}

See the [run](${runUrl}) for details.${migration ? `\n\n${migration}` : ""}${state ? `\n\n\`/revert\` restores \`${state.revertTo}\` as long as production still runs \`${state.tried}\`.` : ""}`;

    await reportToPr(ghClient, prId, `### \`/try\` failed\n\n${summary}`, state).catch(
      (reportError: unknown) => {
        console.error("Failed to report /try failure:", reportError);
      },
    );
    if (checkRunId !== undefined) {
      await completeStatusCheck(ghClient, checkRunId, "failure", {
        title: "/try failed",
        summary,
      });
    }

    throw error;
  }
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
