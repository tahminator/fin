import { GitHubClient } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import {
  DOCKER_HUB_USERNAME,
  DOCKER_REPOSITORY,
  GITHUB_OWNER,
  GITHUB_REPOSITORY,
  K8S_MANIFESTS_REPOSITORY,
  KUSTOMIZATION_FILE_PATH,
} from "../lib/consts";
import { parseCiEnv } from "../lib/env";

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

  await ghClient.updateK8sTagWithPR({
    manifestRepo: [GITHUB_OWNER, K8S_MANIFESTS_REPOSITORY],
    originRepo: [GITHUB_OWNER, GITHUB_REPOSITORY],
    kustomizationFilePath: KUSTOMIZATION_FILE_PATH,
    imageName: `${DOCKER_HUB_USERNAME}/${DOCKER_REPOSITORY}`,
    newTag: releaseTag,
    environment: "production",
  });
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
