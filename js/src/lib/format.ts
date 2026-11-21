const fallbackCurrency = "USD";

const decimalPattern = /^-?\d+(?:\.\d+)?$/u;

function isDecimal(value: string): value is `${number}` {
  return decimalPattern.test(value);
}

export function formatMoney(amount: string, currency: string | null): string {
  if (!isDecimal(amount)) {
    return amount;
  }

  const formatter = new Intl.NumberFormat(undefined, {
    style: "currency",
    currency: currency ?? fallbackCurrency,
  });

  return formatter.format(amount);
}

export function isNegative(amount: string): boolean {
  return amount.startsWith("-");
}

export function formatDate(date: string): string {
  const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

  return formatter.format(new Date(`${date}T00:00:00`));
}

export function formatShortDate(date: string): string {
  const formatter = new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric" });

  return formatter.format(new Date(`${date}T00:00:00`));
}

export function formatDateTime(timestamp: string): string {
  const formatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });

  return formatter.format(new Date(timestamp));
}

const minuteMs = 60_000;

const ageUnits = [
  { suffix: "y", ms: 365 * 24 * 60 * minuteMs },
  { suffix: "w", ms: 7 * 24 * 60 * minuteMs },
  { suffix: "d", ms: 24 * 60 * minuteMs },
  { suffix: "h", ms: 60 * minuteMs },
] as const;

export function formatAge(timestamp: string, now: number): string {
  const elapsed = Math.max(0, now - new Date(timestamp).getTime());

  for (const unit of ageUnits) {
    if (elapsed >= unit.ms) {
      return `${Math.floor(elapsed / unit.ms)}${unit.suffix}`;
    }
  }

  return `${Math.max(1, Math.floor(elapsed / minuteMs))}m`;
}
