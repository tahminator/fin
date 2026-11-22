const scale = 4;

const factor = 10n ** BigInt(scale);

const decimalPattern = /^(?<sign>-?)(?<whole>\d+)(?:\.(?<fraction>\d+))?$/u;

function toScaled(value: string): bigint {
  const groups = decimalPattern.exec(value)?.groups;

  if (groups === undefined) {
    throw new Error(`Not a decimal: ${value}`);
  }

  const { sign = "", whole = "0", fraction = "" } = groups;
  const scaled = BigInt(whole) * factor + BigInt(fraction.padEnd(scale, "0").slice(0, scale));

  return sign === "-" ? -scaled : scaled;
}

function fromScaled(value: bigint): string {
  const negative = value < 0n;
  const absolute = negative ? -value : value;
  const fraction = (absolute % factor).toString().padStart(scale, "0");

  return `${negative ? "-" : ""}${(absolute / factor).toString()}.${fraction}`;
}

export function sumDecimals(values: string[]): string {
  return fromScaled(values.reduce((total, value) => total + toScaled(value), 0n));
}
