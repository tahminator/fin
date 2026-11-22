import { $ } from "bun";

import { MIGRATIONS_DIR } from "./consts";

const VERSION_PATTERN = /^(\d+)( \(dirty\))?$/;

export type SchemaVersion = {
  version: number | undefined;
  dirty: boolean;
};

export function describeSchemaVersion({ version }: SchemaVersion) {
  return version === undefined ? "no migrations" : `version ${version}`;
}

export async function readSchemaVersion(databaseUrl: string): Promise<SchemaVersion> {
  const { exitCode, stderr } =
    await $`migrate -path ${MIGRATIONS_DIR} -database ${databaseUrl} version`.quiet().nothrow();
  const output = stderr.toString().trim();

  if (exitCode !== 0) {
    if (output.endsWith("no migration")) {
      return { version: undefined, dirty: false };
    }

    console.error(output);
    throw new Error("Reading the production schema version failed.");
  }

  const match = VERSION_PATTERN.exec(output);
  if (match?.[1] === undefined) {
    console.error(output);
    throw new Error("Parsing the production schema version failed.");
  }

  return { version: Number(match[1]), dirty: match[2] !== undefined };
}

export async function migrateUp(databaseUrl: string) {
  const { exitCode } = await $`just migrate`
    .env({ ...process.env, DATABASE_URL: databaseUrl })
    .nothrow();

  if (exitCode !== 0) {
    throw new Error(
      `Migrating the production database failed with exit code ${exitCode}, so nothing was deployed. The database may be left dirty at a partially applied migration.`,
    );
  }
}

export async function migrateDown(databaseUrl: string, targetVersion: number) {
  const { exitCode } =
    await $`migrate -path ${MIGRATIONS_DIR} -database ${databaseUrl} goto ${targetVersion}`.nothrow();

  if (exitCode !== 0) {
    throw new Error(
      `Rolling back the production database failed with exit code ${exitCode}. The database may be left dirty at a partially applied migration.`,
    );
  }
}
