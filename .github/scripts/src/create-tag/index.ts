import { GitHubClient, VersioningClient, VersionUpdatingStrategy } from "@tahminator/pipeline";

import { parseCiEnv } from "../lib/env";

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

  const versioningClient = new VersioningClient(ghClient, VersionUpdatingStrategy.NONE);

  await ghClient.createTag({
    nextTag: await versioningClient.next(),
    onPreTagCreate: async (tag) => {
      await versioningClient.update(tag);
    },
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
