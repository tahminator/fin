import { $ } from "bun";

async function main() {
  await $`just test`;
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
