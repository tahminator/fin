import { $ } from "bun";

async function main() {
  await $`pnpm --dir js install --frozen-lockfile`;
  await $`just gen`;
  await $`pnpm --dir js check`;
  await $`pnpm --dir js build`;
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
