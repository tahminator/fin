# AGENTS.md

fin is a personal finance tracker: a Go backend (`./`), a Vite + React PWA (`./js/`), Postgres migrations (`./db/`) and the container build (`./infra/`).

## Read first

- [STYLEGUIDE.md](STYLEGUIDE.md): engineering conventions and hard rules (no comments, natural blank lines, layering, lint, tests). Follow it for every change.
- [agents/plan/](agents/plan): build plans, the source of truth for behavior, endpoints, configuration, milestones and deployment.
  - [init.md](agents/plan/init.md): the base application, CI/CD included.
  - [mcp.md](agents/plan/mcp.md): the MCP server and its OAuth authorization server at `/mcp`.

## Working with the plans

- Check the relevant plan before changing behavior, and update the plan in the same change when behavior deliberately differs from it.
- New features get their own file in `agents/plan/` instead of growing `init.md`. A new plan lists the sections of earlier plans it changes.
- Plans describe intended design; not everything in them is implemented yet.
