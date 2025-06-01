import type { SessionsMap } from "../types";
import { cn } from "../lib/utils";

interface SidebarProps {
  sessions: SessionsMap;
  currentSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
}

export const Sidebar = ({
  sessions,
  currentSessionId,
  onSessionSelect,
}: SidebarProps) => {
  console.log(currentSessionId);
  const sortedSessions = Object.values(sessions);
  return (
    <div className="w-64 border-r bg-background p-4">
      <div className="font-semibold mb-4 text-lg">Sessions</div>
      <div className="space-y-2">
        {sortedSessions
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
                "hover:bg-gray-200",
                session.id === currentSessionId
                  ? "bg-secondary text-secondary-foreground"
                  : "text-foreground"
              )}
              onClick={() => onSessionSelect(session.id)}
            >
              <div className="font-medium">
                Session {session.id.slice(0, 8)}
              </div>
              <div className="text-sm text-muted-foreground">
                {new Date(session.created_at).toLocaleDateString()}
              </div>
            </div>
          ))}
      </div>
    </div>
  );
};
