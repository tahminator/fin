export const GITHUB_OWNER = "tahminator";
export const GITHUB_REPOSITORY = "fin";
export const K8S_MANIFESTS_REPOSITORY = "k8s-personal";
export const DOCKER_HUB_USERNAME = "tahminator";
export const DOCKER_REPOSITORY = "fin";
export const KUSTOMIZATION_FILE_PATH = "base/production/fin/kustomization.yaml";
export const DOCKER_IMAGE = `${DOCKER_HUB_USERNAME}/${DOCKER_REPOSITORY}`;
export const STAGING_TAG_PREFIX = "staging-";
export const IMAGE_WAIT_INTERVAL_MS = 15_000;
export const IMAGE_WAIT_ATTEMPTS = 120;

export function shortSha(sha: string) {
  if (!/^[0-9a-f]{40}$/.test(sha)) {
    throw new Error(`Expected a full commit SHA, got "${sha}"`);
  }
  return sha.slice(0, 7);
}

export function stagingTag(sha: string) {
  return `${STAGING_TAG_PREFIX}${shortSha(sha)}`;
}
