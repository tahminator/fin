export function parseCiEnv<const T extends readonly string[]>(names: T): Record<T[number], string> {
  const result = {} as Record<T[number], string>;
  const missing: string[] = [];

  for (const name of names) {
    const value = process.env[name];

    if (value === undefined || value.length === 0) {
      missing.push(name);
    } else {
      result[name as T[number]] = value;
    }
  }

  if (missing.length > 0) {
    throw new Error(`Missing required CI environment variables: ${missing.join(", ")}`);
  }

  return result;
}
