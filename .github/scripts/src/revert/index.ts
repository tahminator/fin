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

const { prId, author, runUrl } = await yargs(hideBin(process.argv))
  .option("prId", {
    type: "number",
    describe: "Pull request the command was issued from",
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

  try {
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

    await deployToProduction(ghClient, state.revertTo);

    await reportToPr(
      ghClient,
      prId,
      `### Production reverted

@${author} restored \`${DOCKER_IMAGE}:${state.revertTo}\` in production with \`/revert\`, replacing \`${current}\`. ([run](${runUrl}))`,
    );
  } catch (error) {
    await reportToPr(
      ghClient,
      prId,
      `### \`/revert\` failed

${error instanceof Error ? error.message : String(error)}

See the [run](${runUrl}) for details.`,
      state,
    ).catch((reportError: unknown) => {
      console.error("Failed to report /revert failure:", reportError);
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
