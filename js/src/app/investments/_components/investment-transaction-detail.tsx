import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { formatDate, formatDateTime, formatMoney, formatShares } from "@/lib/format";
import type { InvestmentTransaction } from "@/lib/types/api";

function valueOrDash(value: string | null | undefined): string {
  return value === null || value === undefined || value === "" ? "—" : value;
}

function AmountSection({ data }: { data: InvestmentTransaction }) {
  return (
    <ListSection header="Amount">
      <ListRow>
        <ListRowLabel title="Amount" />
        <p className="shrink-0 type-body tabular-nums">{formatMoney(data.amount, data.currency)}</p>
      </ListRow>
    </ListSection>
  );
}

function DetailsSection({ data }: { data: InvestmentTransaction }) {
  return (
    <ListSection header="Details">
      <ListRow>
        <ListRowLabel title="Name" />
        <p className="max-w-[60%] text-right type-body text-muted-foreground">{data.name}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Account" />
        <p className="max-w-[60%] text-right type-body text-muted-foreground">{data.accountName}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Security" />
        <p className="max-w-[60%] text-right type-body text-muted-foreground">
          {valueOrDash(data.securityName)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Ticker" />
        <p className="type-body text-muted-foreground">{valueOrDash(data.ticker)}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Posted" />
        <p className="type-body text-muted-foreground">{formatDate(data.postedOn)}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Transaction time" />
        <p className="type-body text-muted-foreground">
          {data.transactionDatetime === null ? "—" : formatDateTime(data.transactionDatetime)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Type" />
        <p className="type-body text-muted-foreground capitalize">{data.type}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Subtype" />
        <p className="type-body text-muted-foreground capitalize">{data.subtype}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Quantity" />
        <p className="type-body text-muted-foreground tabular-nums">
          {data.quantity === null ? "—" : formatShares(data.quantity)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Price" />
        <p className="type-body text-muted-foreground tabular-nums">
          {data.price === null ? "—" : formatMoney(data.price, data.currency)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Fees" />
        <p className="type-body text-muted-foreground tabular-nums">
          {data.fees === null ? "—" : formatMoney(data.fees, data.currency)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Currency" />
        <p className="type-body text-muted-foreground">{valueOrDash(data.currency)}</p>
      </ListRow>
    </ListSection>
  );
}

function IdentifierSection({ data }: { data: InvestmentTransaction }) {
  return (
    <ListSection header="Identifiers">
      <ListRow>
        <ListRowLabel title="Activity ID" />
        <p className="max-w-[60%] truncate type-caption2 text-muted-foreground">{data.id}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="External ID" />
        <p className="max-w-[60%] truncate type-caption2 text-muted-foreground">
          {data.externalId}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Canceled external ID" />
        <p className="max-w-[60%] truncate type-caption2 text-muted-foreground">
          {valueOrDash(data.cancelExternalId)}
        </p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Account ID" />
        <p className="max-w-[60%] truncate type-caption2 text-muted-foreground">{data.accountId}</p>
      </ListRow>
      <ListRow>
        <ListRowLabel title="Security ID" />
        <p className="max-w-[60%] truncate type-caption2 text-muted-foreground">
          {valueOrDash(data.securityId)}
        </p>
      </ListRow>
    </ListSection>
  );
}

export function InvestmentTransactionDetail({ data }: { data: InvestmentTransaction }) {
  return (
    <>
      <AmountSection data={data} />
      <DetailsSection data={data} />
      <IdentifierSection data={data} />
    </>
  );
}
