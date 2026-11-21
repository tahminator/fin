import { Card, CardContent } from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import type { CurrencyBalance, Summary } from "@/lib/types/api";

function BalanceStat({
  label,
  amount,
  currency,
}: {
  label: string;
  amount: string;
  currency: string;
}) {
  return (
    <div className="rounded-2xl bg-secondary px-3 py-2">
      <p className="type-footnote text-muted-foreground">{label}</p>
      <p className="truncate type-headline tabular-nums">{formatMoney(amount, currency)}</p>
    </div>
  );
}

function CurrencyCard({ balance }: { balance: CurrencyBalance }) {
  return (
    <Card>
      <CardContent className="flex flex-col gap-4">
        <div>
          <p className="type-subheadline text-muted-foreground">Net worth · {balance.currency}</p>
          <p className="truncate type-large-title font-bold tabular-nums">
            {formatMoney(balance.net, balance.currency)}
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <BalanceStat label="Assets" amount={balance.assets} currency={balance.currency} />
          <BalanceStat
            label="Liabilities"
            amount={balance.liabilities}
            currency={balance.currency}
          />
        </div>
      </CardContent>
    </Card>
  );
}

export function BalanceSummary({ summary }: { summary: Summary }) {
  return (
    <section className="flex flex-col gap-1.5">
      <div className="flex flex-col gap-3">
        {summary.balances.map((balance) => (
          <CurrencyCard key={balance.currency} balance={balance} />
        ))}
      </div>
      <p className="px-4 type-footnote text-muted-foreground">
        {summary.accountCount} accounts · {summary.connectionCount} of 10 connections
      </p>
    </section>
  );
}
