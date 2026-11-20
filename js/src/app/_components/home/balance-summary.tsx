import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatMoney } from "@/lib/format";
import type { CurrencyBalance, Summary } from "@/lib/types/api";

function CurrencyCard({ balance }: { balance: CurrencyBalance }) {
  return (
    <Card>
      <CardHeader>
        <CardDescription>Net worth · {balance.currency}</CardDescription>
        <CardTitle className="text-3xl tabular-nums">
          {formatMoney(balance.net, balance.currency)}
        </CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-2 gap-4 text-sm">
        <div>
          <p className="text-muted-foreground">Assets</p>
          <p className="font-medium tabular-nums">
            {formatMoney(balance.assets, balance.currency)}
          </p>
        </div>
        <div>
          <p className="text-muted-foreground">Liabilities</p>
          <p className="font-medium tabular-nums">
            {formatMoney(balance.liabilities, balance.currency)}
          </p>
        </div>
      </CardContent>
    </Card>
  );
}

export function BalanceSummary({ summary }: { summary: Summary }) {
  return (
    <section className="flex flex-col gap-3">
      {summary.balances.map((balance) => (
        <CurrencyCard key={balance.currency} balance={balance} />
      ))}
      <div className="flex items-center gap-2">
        <Badge variant="secondary">{summary.accountCount} accounts</Badge>
        <Badge variant="secondary">{summary.connectionCount} / 10 connections</Badge>
      </div>
    </section>
  );
}
