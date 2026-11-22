import { ListRow, ListRowLabel, ListSection } from "@/components/list-section";
import { Checkbox } from "@/components/ui/checkbox";

interface ScopeListProps {
  syncRequested: boolean;
  allowSync: boolean;
  onAllowSyncChange: (allow: boolean) => void;
}

export function ScopeList({ syncRequested, allowSync, onAllowSyncChange }: ScopeListProps) {
  return (
    <ListSection header="This app will be able to">
      <ListRow>
        <Checkbox checked disabled aria-label="Read access is required" />
        <ListRowLabel title="Read your accounts, balances and transactions" subtitle="Required" />
      </ListRow>
      {syncRequested ? (
        <ListRow>
          <Checkbox
            id="allow-sync"
            checked={allowSync}
            onCheckedChange={(checked) => {
              onAllowSyncChange(checked === true);
            }}
          />
          <label htmlFor="allow-sync" className="min-w-0 flex-1">
            <ListRowLabel
              title="Refresh data from your connected banks"
              subtitle="At most once per bank every few minutes"
            />
          </label>
        </ListRow>
      ) : null}
    </ListSection>
  );
}
