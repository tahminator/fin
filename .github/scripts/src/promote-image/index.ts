import { DockerClient, Utils } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import { DOCKER_HUB_USERNAME, DOCKER_REPOSITORY, shortSha } from "../lib/consts";
import { imageTagExists } from "../lib/docker-hub";
import { parseCiEnv } from "../lib/env";

const IMAGE_WAIT_INTERVAL_MS = 15_000;
const IMAGE_WAIT_ATTEMPTS = 120;

const { releaseTag, sha } = await yargs(hideBin(process.argv))
  .option("releaseTag", {
    type: "string",
    describe: "For example, 1.2.3",
    demandOption: true,
  })
  .option("sha", {
    type: "string",
    describe: "Full SHA of the tagged commit's parent; waits for CI to push its image",
    demandOption: true,
  })
  .strict()
  .parse();

async function main() {
  const { DOCKER_HUB_PAT } = parseCiEnv(["DOCKER_HUB_PAT"]);

  const originalTag = shortSha(sha);

  // the tag is created alongside CI's image build, so the image may still be on its way
  const pushed = await Utils.waitUntil({
    attempts: IMAGE_WAIT_ATTEMPTS,
    intervalMs: IMAGE_WAIT_INTERVAL_MS,
    predicate: async (attempt) => {
      try {
        if (await imageTagExists(originalTag)) {
          return true;
        }
        console.log(
          `Waiting for ${DOCKER_REPOSITORY}:${originalTag} (${attempt}/${IMAGE_WAIT_ATTEMPTS})`,
        );
      } catch (e) {
        console.warn(e);
      }
      return false;
    },
  });
  if (!pushed) {
    throw new Error(
      `${DOCKER_REPOSITORY}:${originalTag} was never pushed; check CI's "Build production image" job on ${sha}`,
    );
  }

  await using dockerClient = await DockerClient.create(DOCKER_HUB_USERNAME, DOCKER_HUB_PAT);

  await dockerClient.promoteDockerImage({
    originalTag,
    newGithubTags: [releaseTag, "latest"],
    repository: DOCKER_REPOSITORY,
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
