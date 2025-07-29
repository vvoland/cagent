import { useState, useCallback, useEffect, useMemo } from "react";
import type { Session, SessionDetail } from "../types";

interface UseSessionsReturn {
  sessions: Session[];
  currentSession: SessionDetail | null;
  currentSessionId: string | null;
  isLoading: boolean;
  error: string | null;
  createNewSession: () => Promise<string | null>;
  selectSession: (sessionId: string) => void;
  deleteSession: (sessionId: string) => Promise<void>;
  refreshSessions: () => Promise<void>;
}

export const useSessions = (): UseSessionsReturn => {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [currentSession, setCurrentSession] = useState<SessionDetail | null>(
    null
  );
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const run = async () => {
      const response = await fetch(`/api/sessions/${currentSessionId}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch session: ${response.statusText}`);
      }
      const data = (await response.json()) as SessionDetail;
      setCurrentSession(data);
    };
    run();
  }, [currentSessionId, sessions]);

  const fetchSessions = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await fetch("/api/sessions");

      if (!response.ok) {
        throw new Error(`Failed to fetch sessions: ${response.statusText}`);
      }

      const data = (await response.json()) as Session[];
      setSessions(data);

      // Only set current session if we don't have one and there are sessions available
      if (data.length > 0 && !currentSessionId) {
        // Get the most recent session
        const mostRecentSession = data.sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )[0];
        if (mostRecentSession) {
          setCurrentSessionId(mostRecentSession.id);
        }
      }
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to fetch sessions";
      console.error("Failed to fetch sessions:", error);
      setError(errorMessage);
      setSessions([]);
    } finally {
      setIsLoading(false);
    }
  }, [currentSessionId]);

  const createNewSession = useCallback(async (): Promise<string | null> => {
    try {
      setError(null);
      const response = await fetch(`/api/sessions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to create session: ${response.statusText}`);
      }

      const newSession = (await response.json()) as Session;
      setSessions((prev) => [...prev, newSession]);
      setCurrentSessionId(newSession.id);
      return newSession.id;
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : "Failed to create new session";
      console.error("Failed to create new session:", error);
      setError(errorMessage);
      return null;
    }
  }, []);

  const selectSession = useCallback((sessionId: string) => {
    setCurrentSessionId(sessionId);
    setError(null); // Clear any existing errors when switching sessions
  }, []);

  const deleteSession = useCallback(
    async (sessionId: string): Promise<void> => {
      try {
        setError(null);
        const response = await fetch(`/api/sessions/${sessionId}`, {
          method: "DELETE",
        });

        if (!response.ok) {
          throw new Error(`Failed to delete session: ${response.statusText}`);
        }

        setSessions((prev) => prev.filter((s) => s.id !== sessionId));

        // If we deleted the current session, select another one or clear it
        if (currentSessionId === sessionId) {
          const remainingSessions = sessions.filter((s) => s.id !== sessionId);
          if (remainingSessions.length > 0) {
            const mostRecentSession = remainingSessions.sort(
              (a, b) =>
                new Date(b.created_at).getTime() -
                new Date(a.created_at).getTime()
            )[0];
            if (mostRecentSession) {
              setCurrentSessionId(mostRecentSession.id);
            } else {
              setCurrentSessionId(null);
            }
          } else {
            setCurrentSessionId(null);
          }
        }
      } catch (error) {
        const errorMessage =
          error instanceof Error ? error.message : "Failed to delete session";
        console.error("Failed to delete session:", error);
        setError(errorMessage);
      }
    },
    [currentSessionId, sessions]
  );

  // Memoize the refresh function to avoid recreation
  const refreshSessions = useCallback(() => fetchSessions(), [fetchSessions]);

  // Fetch sessions on mount
  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  // Memoize the return object to prevent unnecessary re-renders
  const returnValue = useMemo(
    (): UseSessionsReturn => ({
      sessions,
      currentSession,
      currentSessionId,
      isLoading,
      error,
      createNewSession,
      selectSession,
      deleteSession,
      refreshSessions,
    }),
    [
      sessions,
      currentSessionId,
      isLoading,
      error,
      createNewSession,
      selectSession,
      deleteSession,
      refreshSessions,
    ]
  );

  return returnValue;
};
