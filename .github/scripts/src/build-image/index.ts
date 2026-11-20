import { DockerClient, type Environment, GitHubClient } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import {
  DOCKER_HUB_USERNAME,
  DOCKER_REPOSITORY,
  GITHUB_OWNER,
  GITHUB_REPOSITORY,
  shortSha,
} from "../lib/consts";
import { parseCiEnv } from "../lib/env";

const { environment, sha, prId } = await yargs(hideBin(process.argv))
  .option("environment", {
    choices: ["staging", "production"] satisfies Environment[],
    describe: "staging images are tagged staging-<sha>; production are <sha>",
    demandOption: true,
  })
  .option("sha", {
    type: "string",
    describe: "Full commit SHA the image is built from",
    demandOption: true,
  })
  .option("prId", {
    type: "string",
    describe: "Pull request to comment the pushed tags on; empty skips it",
    default: "",
  })
  .strict()
  .parse();

async function main() {
  const {
    DOCKER_HUB_PAT,
    _GITHUB_APP_APP_ID,
    _GITHUB_APP_INSTALLATION_ID,
    _GITHUB_APP_PEM_CONTENT,
  } = parseCiEnv([
    "DOCKER_HUB_PAT",
    "_GITHUB_APP_APP_ID",
    "_GITHUB_APP_INSTALLATION_ID",
    "_GITHUB_APP_PEM_CONTENT",
  ]);

  const image = `${DOCKER_HUB_USERNAME}/${DOCKER_REPOSITORY}`;
  const short = shortSha(sha);
  const tags =
    environment === "staging" ? [`staging-${short}`, `sha-${sha}`] : [short, `sha-${sha}`];

  const goMod = await Bun.file("go.mod").text();
  const { devEngines, packageManager } = await Bun.file("js/package.json").json();

  await using dockerClient = await DockerClient.create(DOCKER_HUB_USERNAME, DOCKER_HUB_PAT);

  await dockerClient.buildImage({
    dockerRepository: DOCKER_REPOSITORY,
    dockerFileLocation: "infra/Dockerfile",
    tags,
    platforms: ["linux/amd64"],
    buildArgs: {
      GO_VERSION: goMod.match(/^go (\S+)$/m)![1],
      NODE_VERSION: devEngines.runtime.version,
      PNPM_VERSION: packageManager.split("@")[1].split("+")[0],
    },
  });

  if (prId) {
    const ghClient = await GitHubClient.createWithGithubAppToken({
      appId: _GITHUB_APP_APP_ID,
      installationId: _GITHUB_APP_INSTALLATION_ID,
      privateKey: _GITHUB_APP_PEM_CONTENT,
    });

    await ghClient.sendPrMessage({
      prId: Number(prId),
      owner: GITHUB_OWNER,
      repository: GITHUB_REPOSITORY,
      message: `The fin image has been uploaded to https://hub.docker.com/r/${image}/tags under the following tags:

${tags.map((t) => `- \`${DOCKER_REPOSITORY}:${t}\``).join("\n")}
`,
    });
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
