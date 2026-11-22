interface SecurityIdentity {
  id: string;
  ticker: string | null;
  name: string | null;
}

function present(value: string | null): value is string {
  return value !== null && value.trim() !== "";
}

export function securityLabel({ id, ticker, name }: SecurityIdentity): string {
  if (present(ticker) && present(name)) {
    return `${ticker} · ${name}`;
  }

  if (present(ticker)) {
    return ticker;
  }

  if (present(name)) {
    return name;
  }

  return `Unnamed security ${id.slice(-4)}`;
}
