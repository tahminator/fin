import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { formatDate, formatMoney } from "@/lib/format";
import type { TransactionDetail } from "@/lib/types/api";

function valueOrDash(value: string | null | undefined): string {
  return value === null || value === undefined || value === "" ? "—" : value;
}

export function TransactionDetailView({ data }: { data: TransactionDetail }) {
  return (
    <>
      <ListSection header="Amount">
        <ListRow>
          <ListRowLabel title="Amount" />
          <p className="shrink-0 type-body tabular-nums">
            {formatMoney(data.amount, data.currency)}
          </p>
        </ListRow>
      </ListSection>
      <ListSection header="Details">
        <ListRow>
          <ListRowLabel title="Name" />
          <p className="max-w-[60%] text-right type-body text-muted-foreground">{data.name}</p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Merchant" />
          <p className="max-w-[60%] text-right type-body text-muted-foreground">
            {valueOrDash(data.merchantName)}
          </p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Account" />
          <p className="max-w-[60%] text-right type-body text-muted-foreground">
            {data.accountName}
          </p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Posted" />
          <p className="type-body text-muted-foreground">{formatDate(data.postedOn)}</p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Authorized" />
          <p className="type-body text-muted-foreground">
            {data.authorizedOn === null ? "—" : formatDate(data.authorizedOn)}
          </p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Category" />
          <p className="max-w-[60%] text-right type-body text-muted-foreground">
            {valueOrDash(data.category)}
          </p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Status" />
          <p className="type-body text-muted-foreground">{data.pending ? "Pending" : "Posted"}</p>
        </ListRow>
      </ListSection>
      <ListSection header="Identifiers">
        <ListRow>
          <ListRowLabel title="Transaction ID" />
          <p className="max-w-[60%] truncate type-caption1 text-muted-foreground">{data.id}</p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="External ID" />
          <p className="max-w-[60%] truncate type-caption1 text-muted-foreground">
            {data.externalId}
          </p>
        </ListRow>
        <ListRow>
          <ListRowLabel title="Account ID" />
          <p className="max-w-[60%] truncate type-caption1 text-muted-foreground">
            {data.accountId}
          </p>
        </ListRow>
      </ListSection>
    </>
  );
}
