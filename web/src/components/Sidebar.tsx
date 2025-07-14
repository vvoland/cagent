import type { Session } from "../types";
import { cn } from "../lib/utils";

interface SidebarProps {
  sessions: Session[];
  currentSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
}

export const Sidebar = ({
  sessions,
  currentSessionId,
  onSessionSelect,
}: SidebarProps) => {
  return (
    <div className="w-64 border-r bg-background p-4 dark:border-border dark:bg-background">
      <div className="font-semibold mb-4 text-lg dark:text-foreground">
        Sessions
      </div>
      <div className="space-y-2">
        {sessions
          .sort(
            (a, b) =>
              new Date(b.created_at).getTime() -
              new Date(a.created_at).getTime()
          )
          .map((session) => (
            <div
              key={session.id}
              className={cn(
                "p-3 rounded-lg cursor-pointer transition-colors",
                "hover:bg-gray-200 dark:hover:bg-gray-700",
                session.id === currentSessionId
                  ? "bg-secondary text-secondary-foreground dark:bg-secondary dark:text-secondary-foreground"
                  : "text-foreground dark:text-foreground"
              )}
              onClick={() => onSessionSelect(session.id)}
            >
              <div className="font-medium dark:text-foreground">
                Session {session.id.slice(0, 8)}
              </div>
              <div className="text-sm text-muted-foreground dark:text-muted-foreground">
                {new Date(session.created_at).toLocaleDateString()}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
};
