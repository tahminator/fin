import { ListFilterIcon } from "lucide-react";
import { useSearchParams } from "react-router";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useAllAccounts } from "@/lib/queries/accounts";
import type { Account } from "@/lib/types/api";

interface ConnectionGroup {
  connectionId: string;
  institutionName: string;
  accounts: Account[];
}

function groupByConnection(accounts: Account[]): ConnectionGroup[] {
  const groups = new Map<string, ConnectionGroup>();

  for (const account of accounts) {
    const group = groups.get(account.connectionId);

    if (group === undefined) {
      groups.set(account.connectionId, {
        connectionId: account.connectionId,
        institutionName: account.institutionName ?? "Other",
        accounts: [account],
      });
    } else {
      group.accounts.push(account);
    }
  }

  return [...groups.values()];
}

function ConnectionGroupFields({
  group,
  hidden,
  onToggleAccount,
  onToggleGroup,
}: {
  group: ConnectionGroup;
  hidden: Set<string>;
  onToggleAccount: (accountId: string, show: boolean) => void;
  onToggleGroup: (group: ConnectionGroup, show: boolean) => void;
}) {
  const hiddenInGroup = group.accounts.filter((account) => hidden.has(account.id)).length;
  const groupChecked = hiddenInGroup === 0;
  const groupIndeterminate = hiddenInGroup > 0 && hiddenInGroup < group.accounts.length;

  return (
    <div className="flex flex-col gap-1.5">
      <label className="flex items-center gap-2 text-sm font-medium">
        <Checkbox
          checked={groupIndeterminate ? "indeterminate" : groupChecked}
          onCheckedChange={(checked) => {
            onToggleGroup(group, checked === true);
          }}
        />
        {group.institutionName}
      </label>
      <div className="flex flex-col gap-1 pl-6">
        {group.accounts.map((account) => (
          <label key={account.id} className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={!hidden.has(account.id)}
              onCheckedChange={(checked) => {
                onToggleAccount(account.id, checked === true);
              }}
            />
            {account.name}
            {account.mask === null ? null : ` ••${account.mask}`}
          </label>
        ))}
      </div>
    </div>
  );
}

export function TransactionsFilter() {
  const [searchParams, setSearchParams] = useSearchParams();
  const accounts = useAllAccounts();

  const hidden = new Set(searchParams.getAll("hide"));

  function setHidden(next: Set<string>) {
    const params = new URLSearchParams(searchParams);

    params.delete("hide");

    for (const accountId of next) {
      params.append("hide", accountId);
    }

    setSearchParams(params);
  }

  function toggleAccount(accountId: string, show: boolean) {
    const next = new Set(hidden);

    if (show) {
      next.delete(accountId);
    } else {
      next.add(accountId);
    }

    setHidden(next);
  }

  function toggleGroup(group: ConnectionGroup, show: boolean) {
    const next = new Set(hidden);

    for (const account of group.accounts) {
      if (show) {
        next.delete(account.id);
      } else {
        next.add(account.id);
      }
    }

    setHidden(next);
  }

  if (accounts.data === undefined) {
    return (
      <Button variant="outline" size="sm" disabled>
        <ListFilterIcon />
        Filter
      </Button>
    );
  }

  const groups = groupByConnection(accounts.data);
  const hiddenCount = accounts.data.filter((account) => hidden.has(account.id)).length;

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm">
          <ListFilterIcon />
          {hiddenCount === 0 ? "Filter" : `Filter (${hiddenCount} hidden)`}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80">
        <div className="flex items-center justify-between">
          <p className="font-medium">Accounts</p>
          <Button
            variant="ghost"
            size="xs"
            onClick={() => {
              setHidden(
                hiddenCount === 0 ? new Set(accounts.data.map((account) => account.id)) : new Set(),
              );
            }}
          >
            {hiddenCount === 0 ? "Hide all" : "Show all"}
          </Button>
        </div>
        <div className="flex max-h-80 flex-col gap-3 overflow-y-auto">
          {groups.map((group) => (
            <ConnectionGroupFields
              key={group.connectionId}
              group={group}
              hidden={hidden}
              onToggleAccount={toggleAccount}
              onToggleGroup={toggleGroup}
            />
          ))}
        </div>
      </PopoverContent>
    </Popover>
  );
}
