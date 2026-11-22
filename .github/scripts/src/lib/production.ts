import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

import type { GitHubClient } from "@tahminator/pipeline";
import { $ } from "bun";

import {
  DOCKER_IMAGE,
  GITHUB_OWNER,
  GITHUB_REPOSITORY,
  K8S_MANIFESTS_REPOSITORY,
  KUSTOMIZATION_FILE_PATH,
} from "./consts";

const PR_MESSAGE_KEY = "production-try";
const TRY_STATE_PATTERN = /<!-- fin-try-state:(.+?) -->/;

export type TryState = {
  tried: string;
  revertTo: string;
  revertSchemaVersion: number | null;
  triedSchemaVersion: number | null;
};

export async function deployToProduction(ghClient: GitHubClient, tag: string) {
  await ghClient.updateK8sTagWithPR({
    manifestRepo: [GITHUB_OWNER, K8S_MANIFESTS_REPOSITORY],
    originRepo: [GITHUB_OWNER, GITHUB_REPOSITORY],
    kustomizationFilePath: KUSTOMIZATION_FILE_PATH,
    imageName: DOCKER_IMAGE,
    newTag: tag,
    environment: "production",
  });
}

/**
 * Reads the tag production runs right now from the manifests repository.
 * TODO: Move this tag reading to pipeline library
 */
export async function readProductionTag(): Promise<string> {
  const dir = await mkdtemp(join(tmpdir(), `${K8S_MANIFESTS_REPOSITORY}-`));

  try {
    await $`git clone --quiet --depth=1 --filter=blob:none --no-checkout https://github.com/${GITHUB_OWNER}/${K8S_MANIFESTS_REPOSITORY}.git ${dir}`;

    const { images } = Bun.YAML.parse(
      await $`git -C ${dir} show HEAD:${KUSTOMIZATION_FILE_PATH}`.text(),
    ) as { images?: { name: string; newTag?: unknown }[] };

    const tag = images?.find((image) => image.name === DOCKER_IMAGE)?.newTag;
    if (typeof tag !== "string") {
      throw new Error(
        `Expected a string ${DOCKER_IMAGE} newTag in ${KUSTOMIZATION_FILE_PATH}, got ${JSON.stringify(tag)}`,
      );
    }
    return tag;
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

export async function readTryState(
  ghClient: GitHubClient,
  prId: number,
): Promise<TryState | undefined> {
  const comment = await ghClient.getPrMessage({
    prId,
    owner: GITHUB_OWNER,
    repository: GITHUB_REPOSITORY,
    key: PR_MESSAGE_KEY,
  });

  const json = comment?.body?.match(TRY_STATE_PATTERN)?.[1];
  if (json === undefined) {
    return undefined;
  }

  const state = JSON.parse(json);
  if (
    typeof state?.tried !== "string" ||
    typeof state?.revertTo !== "string" ||
    (typeof state?.revertSchemaVersion !== "number" && state?.revertSchemaVersion !== null) ||
    (typeof state?.triedSchemaVersion !== "number" && state?.triedSchemaVersion !== null)
  ) {
    throw new Error(`Malformed /try state on PR #${prId}: ${json}`);
  }
  return {
    tried: state.tried,
    revertTo: state.revertTo,
    revertSchemaVersion: state.revertSchemaVersion,
    triedSchemaVersion: state.triedSchemaVersion,
  };
}

/**
 * Upserts this PR's production comment. `state` is embedded as a hidden marker for `/revert`;
 * omitting it clears whatever the comment previously recorded.
 */
export async function reportToPr(
  ghClient: GitHubClient,
  prId: number,
  message: string,
  state?: TryState,
) {
  await ghClient.sendPrMessage({
    prId,
    owner: GITHUB_OWNER,
    repository: GITHUB_REPOSITORY,
    message: state ? `${message}\n\n<!-- fin-try-state:${JSON.stringify(state)} -->` : message,
    key: PR_MESSAGE_KEY,
  });
}
