import { memo, useCallback, useMemo } from "react";
import type { Session } from "../types";
import { cn } from "../lib/utils";
import { Button } from "./ui/button";
import { Trash2, MessageSquare } from "lucide-react";

interface SidebarProps {
  sessions: Session[];
  currentSessionId: string | null;
  onSessionSelect: (sessionId: string) => void;
  onDeleteSession: (sessionId: string) => void;
}

interface SessionItemProps {
  session: Session;
  isActive: boolean;
  onSelect: () => void;
  onDelete: () => void;
}

const SessionItem = memo<SessionItemProps>(({ session, isActive, onSelect, onDelete }) => {
  const handleDeleteClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
    onDelete();
  }, [onDelete]);

  const formattedDate = useMemo(() => {
    const date = new Date(session.created_at);
    const now = new Date();
    const diffInHours = Math.abs(now.getTime() - date.getTime()) / (1000 * 60 * 60);
    
    if (diffInHours < 24) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } else if (diffInHours < 24 * 7) {
      return date.toLocaleDateString([], { weekday: 'short', hour: '2-digit', minute: '2-digit' });
    } else {
      return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
    }
  }, [session.created_at]);

  const sessionId = session.id.slice(0, 8);
  const sessionName =
    session.messages && session.messages[0] && session.messages[0].message.content
      ? session.messages[0].message.content.slice(0, 50)
      : "Untitled Session";
  const messageCount = session.messages?.length || 0;

  return (
    <div
      className={cn(
        "p-3 rounded-lg cursor-pointer transition-all relative group",
        "hover:bg-gray-200/80 dark:hover:bg-gray-700/80 hover:shadow-sm",
        "border border-transparent hover:border-primary/20",
        isActive
          ? "bg-secondary text-secondary-foreground dark:bg-secondary dark:text-secondary-foreground shadow-sm border-primary/30"
          : "text-foreground dark:text-foreground"
      )}
      onClick={onSelect}
      role="button"
      tabIndex={0}
      aria-label={`Session ${sessionId} with ${messageCount} messages`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect();
        }
      }}
    >
      <div className="flex items-start justify-between mb-1">
        <div className="font-medium dark:text-foreground flex items-center gap-2 min-w-0">
          <MessageSquare className="h-4 w-4 flex-shrink-0 text-muted-foreground" />
          <span className="truncate">{sessionName}</span>
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="opacity-0 group-hover:opacity-100 transition-all p-1 h-6 w-6 hover:bg-destructive/10 hover:text-destructive flex-shrink-0"
          onClick={handleDeleteClick}
          aria-label={`Delete session ${sessionId}`}
        >
          <Trash2 className="h-3 w-3" />
        </Button>
      </div>
      
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>{formattedDate}</span>
        {messageCount > 0 && (
          <span className="bg-secondary px-1.5 py-0.5 rounded-full text-xs">
            {messageCount} msg{messageCount !== 1 ? 's' : ''}
          </span>
        )}
      </div>
    </div>
  );
});

SessionItem.displayName = 'SessionItem';

export const Sidebar = memo<SidebarProps>(({
  sessions,
  currentSessionId,
  onSessionSelect,
  onDeleteSession,
}) => {
  // Memoize sorted sessions for better performance
  const sortedSessions = useMemo(() => 
    [...sessions].sort(
      (a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    ), 
    [sessions]
  );

  const handleSessionSelect = useCallback((sessionId: string) => {
    onSessionSelect(sessionId);
  }, [onSessionSelect]);

  const handleDeleteSession = useCallback((sessionId: string) => {
    onDeleteSession(sessionId);
  }, [onDeleteSession]);

  return (
    <aside className="w-64 h-screen border-r bg-background/95 backdrop-blur-sm p-4 dark:border-border dark:bg-background/95 flex flex-col">
      <header className="font-semibold mb-4 text-lg dark:text-foreground flex items-center gap-2">
        <MessageSquare className="h-5 w-5 text-primary" />
        <span>Sessions</span>
        {sessions.length > 0 && (
          <span className="text-xs bg-secondary px-2 py-1 rounded-full text-muted-foreground">
            {sessions.length}
          </span>
        )}
      </header>
      
      <div className="flex-1 overflow-y-auto min-h-0">
        {sortedSessions.length === 0 ? (
          <div className="text-center text-muted-foreground py-8">
            <MessageSquare className="h-12 w-12 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No sessions yet</p>
            <p className="text-xs mt-1">Create a new session to get started</p>
          </div>
        ) : (
          <div className="space-y-2">
            {sortedSessions.map((session) => (
              <SessionItem
                key={session.id}
                session={session}
                isActive={session.id === currentSessionId}
                onSelect={() => handleSessionSelect(session.id)}
                onDelete={() => handleDeleteSession(session.id)}
              />
            ))}
          </div>
        )}
      </div>
    </aside>
  );
});

Sidebar.displayName = 'Sidebar';