import { Trash2Icon } from "lucide-react";
import { toast } from "sonner";

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { errorMessage } from "@/lib/errors";
import { useRemoveConnection } from "@/lib/queries/connections";

interface RemoveConnectionButtonProps {
  connectionId: string;
  name: string;
}

export function RemoveConnectionButton({ connectionId, name }: RemoveConnectionButtonProps) {
  const remove = useRemoveConnection();

  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button
          size="icon-sm"
          variant="ghost"
          className="ml-auto text-destructive"
          disabled={remove.isPending}
          aria-label={`Remove ${name}`}
        >
          {remove.isPending ? <Spinner /> : <Trash2Icon />}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Remove {name}?</AlertDialogTitle>
          <AlertDialogDescription>
            This disconnects the bank and deletes its accounts and transactions from fin.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => {
              toast.promise(remove.mutateAsync({ params: { path: { connectionId } } }), {
                loading: `Removing ${name}…`,
                success: `${name} removed`,
                error: (error: unknown) => errorMessage(error),
              });
            }}
          >
            Remove
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
