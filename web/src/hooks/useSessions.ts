import { useState, useCallback, useEffect } from "react";
import type { SessionsMap, Session } from "../types";

export const useSessions = () => {
  const [sessions, setSessions] = useState<SessionsMap>({});
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null);

  const fetchSessions = useCallback(async () => {
    try {
      const response = await fetch("/api/sessions");
      const data = (await response.json()) as SessionsMap;
      setSessions(data);
      if (Object.keys(data).length > 0 && !currentSessionId) {
        // Get the most recent session
        const sessions = Object.values(data) as Session[];
        const mostRecentSession = sessions.sort(
          (a, b) =>
            new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )[0];
        setCurrentSessionId(mostRecentSession.id);
      }
    } catch (error) {
      console.error("Failed to fetch sessions:", error);
    }
  }, [currentSessionId]);

  const createNewSession = async (agent: string) => {
    try {
      const response = await fetch(`/api/sessions/${agent}`, {
        method: "POST",
      });
      const newSession = await response.json();
      setSessions((prev) => ({
        ...prev,
        [newSession.id]: newSession,
      }));
      setCurrentSessionId(newSession.id);
      return newSession.id;
    } catch (error) {
      console.error("Failed to create new session:", error);
      return null;
    }
  };

  const selectSession = (sessionId: string) => {
    setCurrentSessionId(sessionId);
  };

  // Fetch sessions on mount
  useEffect(() => {
    fetchSessions();
  }, [fetchSessions]);

  return {
    sessions,
    currentSessionId,
    createNewSession,
    selectSession,
  };
};
