import { TriangleAlertIcon } from "lucide-react";
import { Link } from "react-router";

import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { useNotifications } from "@/lib/queries/notifications";

export function ReauthBanner() {
  const notifications = useNotifications();

  if (!notifications.isSuccess || notifications.data.items.length === 0) {
    return null;
  }

  return (
    <Alert variant="destructive">
      <TriangleAlertIcon />
      <AlertTitle>Sign-in needed</AlertTitle>
      <AlertDescription>
        {notifications.data.items.map((notification) => (
          <p key={notification.connectionId}>
            <strong>{notification.institutionName}</strong> needs you to sign in again
          </p>
        ))}
      </AlertDescription>
      <AlertAction>
        <Button size="sm" variant="outline" asChild>
          <Link to="/connections">Fix</Link>
        </Button>
      </AlertAction>
    </Alert>
  );
}
