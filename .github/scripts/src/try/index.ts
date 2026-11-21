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
import {
  deployToProduction,
  readProductionTag,
  readTryState,
  reportToPr,
  type TryState,
} from "../lib/production";

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
  const { _GITHUB_APP_APP_ID, _GITHUB_APP_INSTALLATION_ID, _GITHUB_APP_PEM_CONTENT } = parseCiEnv([
    "_GITHUB_APP_APP_ID",
    "_GITHUB_APP_INSTALLATION_ID",
    "_GITHUB_APP_PEM_CONTENT",
  ]);

  const ghClient = await GitHubClient.createWithGithubAppToken({
    appId: _GITHUB_APP_APP_ID,
    installationId: _GITHUB_APP_INSTALLATION_ID,
    privateKey: _GITHUB_APP_PEM_CONTENT,
  });

  const tag = stagingTag(sha);
  let state: TryState | undefined;

  try {
    state = await readTryState(ghClient, prId);

    const pushed = await Utils.waitUntil({
      attempts: IMAGE_WAIT_ATTEMPTS,
      intervalMs: IMAGE_WAIT_INTERVAL_MS,
      predicate: async (attempt) => {
        try {
          if (await imageTagExists(tag)) {
            return true;
          }
          console.log(`Waiting for ${DOCKER_IMAGE}:${tag} (${attempt}/${IMAGE_WAIT_ATTEMPTS})`);
        } catch (e) {
          console.warn(e);
        }
        return false;
      },
    });
    if (!pushed) {
      throw new Error(
        `${DOCKER_IMAGE}:${tag} was never pushed; check CI's "Build staging image" job on ${sha}, then comment /try again.`,
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

    if (current !== tag) {
      await deployToProduction(ghClient, tag);
    }
    state = { tried: tag, revertTo };

    await reportToPr(
      ghClient,
      prId,
      `### Production is running this PR

@${author} deployed \`${DOCKER_IMAGE}:${tag}\` (${sha}) to production with \`/try\`, replacing \`${current}\`. ([run](${runUrl}))

Comment \`/revert\` to restore \`${revertTo}\`. Database migrations from this PR were not applied.`,
      state,
    );
  } catch (error) {
    await reportToPr(
      ghClient,
      prId,
      `### \`/try\` failed

${error instanceof Error ? error.message : String(error)}

See the [run](${runUrl}) for details.${state ? `\n\n\`/revert\` restores \`${state.revertTo}\` as long as production still runs \`${state.tried}\`.` : ""}`,
      state,
    ).catch((reportError: unknown) => {
      console.error("Failed to report /try failure:", reportError);
    });

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
