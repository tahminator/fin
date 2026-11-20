// each command dispatches `.github/workflows/${name}-command.yml`, which must accept the
// prId, repository, comment-id, and author inputs.
//
// https://github.com/peter-evans/slash-command-dispatch/blob/main/docs/workflow-dispatch.md
export const COMMANDS = ["try", "revert"] as const;
