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
