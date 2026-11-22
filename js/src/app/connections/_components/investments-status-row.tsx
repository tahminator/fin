import { InvestmentsConsentButton } from "@/app/connections/_components/investments-consent-button";
import { ListRow, ListRowLabel } from "@/components/list-section";
import type { Connection, InvestmentsStatus } from "@/lib/types/api";

const statusText: Record<
  Exclude<InvestmentsStatus, "ready">,
  { title: string; subtitle: string }
> = {
  consent_required: {
    title: "Investments not shared",
    subtitle: "Share investment data to see holdings and portfolio value.",
  },
  pending: {
    title: "Investments are being prepared",
    subtitle: "Your bank is still preparing investment history. It will appear after a later sync.",
  },
  not_enabled: {
    title: "Investments not enabled",
    subtitle:
      "Enable the Investments product for your Plaid client in the Plaid Dashboard, then sync.",
  },
  unsupported: {
    title: "Investments unavailable",
    subtitle: "This institution does not share investment holdings through Plaid.",
  },
};

export function InvestmentsStatusRow({ connection }: { connection: Connection }) {
  const status = connection.investmentsStatus;

  if (status === null || status === "ready") {
    return null;
  }

  const text = statusText[status];

  return (
    <ListRow className="not-first:before:left-4">
      <ListRowLabel
        title={<span className="whitespace-normal">{text.title}</span>}
        subtitle={<span className="whitespace-normal">{text.subtitle}</span>}
      />
      {status === "consent_required" && connection.status === "active" ? (
        <InvestmentsConsentButton connectionId={connection.id} />
      ) : null}
    </ListRow>
  );
}
