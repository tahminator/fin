import { DOCKER_HUB_USERNAME, DOCKER_REPOSITORY } from "./consts";

export async function imageTagExists(tag: string) {
  const res = await fetch(
    `https://hub.docker.com/v2/namespaces/${DOCKER_HUB_USERNAME}/repositories/${DOCKER_REPOSITORY}/tags/${encodeURIComponent(tag)}`,
  );

  if (res.status === 404) {
    return false;
  }
  if (!res.ok) {
    throw new Error(
      `Docker Hub returned ${res.status} while looking up ${DOCKER_REPOSITORY}:${tag}`,
    );
  }
  return true;
}
