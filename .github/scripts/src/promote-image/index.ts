import { DockerClient } from "@tahminator/pipeline";
import yargs from "yargs";
import { hideBin } from "yargs/helpers";

import { DOCKER_HUB_USERNAME, DOCKER_REPOSITORY, shortSha } from "../lib/consts";
import { parseCiEnv } from "../lib/env";

const { releaseTag, sha } = await yargs(hideBin(process.argv))
  .option("releaseTag", {
    type: "string",
    describe: "For example, 1.2.3",
    demandOption: true,
  })
  .option("sha", {
    type: "string",
    describe: "Full SHA of the tagged commit's parent; its image must already exist",
    demandOption: true,
  })
  .strict()
  .parse();

async function main() {
  const { DOCKER_HUB_PAT } = parseCiEnv(["DOCKER_HUB_PAT"]);

  await using dockerClient = await DockerClient.create(DOCKER_HUB_USERNAME, DOCKER_HUB_PAT);

  await dockerClient.promoteDockerImage({
    originalTag: shortSha(sha),
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
