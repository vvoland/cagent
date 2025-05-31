import type { SessionsMap } from "../types";

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
  const sortedSessions = Object.values(sessions);
  return (
    <div className="sidebar">
      {sortedSessions
        .sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
        .map((session) => (
          <div
            key={session.id}
            className={`session-item ${
              session.id === currentSessionId ? "active" : ""
            }`}
            onClick={() => onSessionSelect(session.id)}
          >
            Session {session.id.slice(0, 8)}
            <div className="session-date">
              {new Date(session.created_at).toLocaleDateString()}
            </div>
          </div>
        ))}
    </div>
  );
};
