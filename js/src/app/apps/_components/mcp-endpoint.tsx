import { CopyIcon } from "lucide-react";
import { toast } from "sonner";

import { ListRow, ListSection } from "@/components/list-section";
import { Button } from "@/components/ui/button";

async function copyEndpoint(endpoint: string) {
  try {
    await navigator.clipboard.writeText(endpoint);
    toast.success("Copied");
  } catch {
    toast.error("Couldn’t copy. Select the URL instead.");
  }
}

export function McpEndpoint() {
  const endpoint = `${globalThis.location.origin}/mcp`;

  return (
    <ListSection
      header="MCP server"
      footer={
        <>
          In Claude, add a custom connector with this URL. In Claude Code, run{" "}
          <code className="break-all">claude mcp add --transport http fin {endpoint}</code>.
        </>
      }
    >
      <ListRow>
        <code className="min-w-0 flex-1 truncate type-body select-all">{endpoint}</code>
        <Button
          size="icon-sm"
          variant="ghost"
          aria-label="Copy MCP server URL"
          onClick={() => {
            void copyEndpoint(endpoint);
          }}
        >
          <CopyIcon />
        </Button>
      </ListRow>
    </ListSection>
  );
}
