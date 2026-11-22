import { InfoIcon } from "lucide-react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import type { PortfolioPerformance } from "@/lib/types/api";

const notices = {
  not_configured: {
    title: "Market prices are off",
    description:
      "Charts use your brokerage's latest prices until an Alpaca market data key is added for your account.",
  },
  unavailable: {
    title: "Market prices are unavailable",
    description: "Alpaca didn't respond, so this shows your brokerage's latest prices instead.",
  },
} as const;

export function MarketDataNotice({ status }: { status: PortfolioPerformance["marketData"] }) {
  if (status === "live") {
    return null;
  }

  const notice = notices[status];

  return (
    <Alert>
      <InfoIcon />
      <AlertTitle>{notice.title}</AlertTitle>
      <AlertDescription>{notice.description}</AlertDescription>
    </Alert>
  );
}
