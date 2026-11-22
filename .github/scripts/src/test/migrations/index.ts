import { LocalPostgresClient } from "@tahminator/pipeline";
import { $ } from "bun";

async function main() {
  await using pgClient = await LocalPostgresClient.create({
    database: "fin",
    version: 19,
  });

  const { database, host, port, password, user } = pgClient.state;
  const databaseUrl = new URL(`postgresql://${host}:${port}/${database}`);
  databaseUrl.username = user;
  databaseUrl.password = password;
  databaseUrl.searchParams.set("sslmode", "disable");

  await $.env({
    ...process.env,
    DATABASE_URL: databaseUrl.toString(),
  })`just migrate && just migrate && just migrate-down 1 && just migrate && just migrate-down -all && just migrate && just test-db`;
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((e) => {
    console.error(e);
    process.exit(1);
  });
