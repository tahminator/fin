import { GitHubClient } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import { parseCiEnv } from "../lib/env";
import { deployToProduction } from "../lib/production";

const { releaseTag } = await yargs(hideBin(process.argv))
  .option("releaseTag", {
    type: "string",
    describe: "Deploy the version named by a release tag, for example 1.2.3",
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

  await deployToProduction(ghClient, releaseTag);
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
