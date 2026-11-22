import { GitHubClient } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import { DOCKER_IMAGE } from "../lib/consts";
import { parseCiEnv } from "../lib/env";
import {
  deployToProduction,
  readProductionTag,
  readTryState,
  reportToPr,
  type TryState,
} from "../lib/production";
import { completeStatusCheck, startStatusCheck, updateStatusCheck } from "../lib/status-check";

const CHECK_NAME = "Revert production";

const { prId, sha, author, runUrl } = await yargs(hideBin(process.argv))
  .option("prId", {
    type: "number",
    describe: "Pull request the command was issued from",
    demandOption: true,
  })
  .option("sha", {
    type: "string",
    describe: "Full SHA of the pull request head; the status check is published on it",
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

  let state: TryState | undefined;
  let checkRunId: number | undefined;

  try {
    checkRunId = await startStatusCheck(ghClient, {
      name: CHECK_NAME,
      sha,
      detailsUrl: runUrl,
      output: {
        title: "Checking production",
        summary: "Checking that production still runs this PR's `/try` image.",
      },
    });

    state = await readTryState(ghClient, prId);
    if (state === undefined) {
      throw new Error("This PR has no live `/try` to revert.");
    }

    const current = await readProductionTag();
    if (current !== state.tried) {
      throw new Error(
        `Production now runs \`${current}\`, not this PR's \`${state.tried}\`, so restoring \`${state.revertTo}\` would overwrite a newer deploy.`,
      );
    }

    await updateStatusCheck(ghClient, checkRunId, {
      title: "Restoring production",
      summary: `Restoring \`${DOCKER_IMAGE}:${state.revertTo}\` in production, replacing \`${current}\`.`,
    });

    await deployToProduction(ghClient, state.revertTo);

    const summary = `@${author} restored \`${DOCKER_IMAGE}:${state.revertTo}\` in production with \`/revert\`, replacing \`${current}\`. ([run](${runUrl}))

Database migrations applied by \`/try\` were not rolled back.`;

    await reportToPr(ghClient, prId, `### Production reverted\n\n${summary}`);
    await completeStatusCheck(ghClient, checkRunId, "success", {
      title: "Production reverted",
      summary,
    });
  } catch (error) {
    const summary = `${error instanceof Error ? error.message : String(error)}

See the [run](${runUrl}) for details.`;

    await reportToPr(ghClient, prId, `### \`/revert\` failed\n\n${summary}`, state).catch(
      (reportError: unknown) => {
        console.error("Failed to report /revert failure:", reportError);
      },
    );
    if (checkRunId !== undefined) {
      await completeStatusCheck(ghClient, checkRunId, "failure", {
        title: "/revert failed",
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
