import { $ } from "bun";

async function main() {
  await $`just lint-go`;
  await $`golangci-lint fmt --diff ./...`;
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
