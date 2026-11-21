import type { GitHubClient } from "@tahminator/pipeline";

import { GITHUB_OWNER, GITHUB_REPOSITORY } from "./consts";

type StatusCheckOutput = {
  title: string;
  summary: string;
};

export async function startStatusCheck(
  ghClient: GitHubClient,
  {
    name,
    sha,
    detailsUrl,
    output,
  }: {
    name: string;
    sha: string;
    detailsUrl: string;
    output: StatusCheckOutput;
  },
): Promise<number> {
  const checkRun = await ghClient.statusCheck({
    action: "create",
    owner: GITHUB_OWNER,
    repository: GITHUB_REPOSITORY,
    sha,
    name,
    status: "in_progress",
    output,
    detailsUrl,
  });
  if (checkRun === null) {
    throw new Error(`GitHub did not return the "${name}" check run it created on ${sha}`);
  }

  return checkRun.id;
}

export async function updateStatusCheck(
  ghClient: GitHubClient,
  checkRunId: number,
  output: StatusCheckOutput,
) {
  await ghClient.statusCheck({
    action: "update",
    owner: GITHUB_OWNER,
    repository: GITHUB_REPOSITORY,
    checkRunId,
    output,
  });
}

/** Completion is best-effort: a reporting failure must not turn a finished deploy into an error. */
export async function completeStatusCheck(
  ghClient: GitHubClient,
  checkRunId: number,
  conclusion: "success" | "failure",
  output: StatusCheckOutput,
) {
  await ghClient
    .statusCheck({
      action: "update",
      owner: GITHUB_OWNER,
      repository: GITHUB_REPOSITORY,
      checkRunId,
      status: "completed",
      conclusion,
      output,
    })
    .catch((error: unknown) => {
      console.error(`Failed to mark check run ${checkRunId} as ${conclusion}:`, error);
    });
}
