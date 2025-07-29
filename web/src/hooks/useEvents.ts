import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import type { EventItem, Event, AgentMessage, SessionDetail } from "../types";

interface UseEventsReturn {
  events: EventItem[];
  isLoading: boolean;
  error: string | null;
  handleSubmit: (sessionId: string, prompt: string) => Promise<void>;
}

export const useEvents = (
  session: SessionDetail | null,
  selectedAgent: string | null,
  refreshSessions?: () => Promise<void>
): UseEventsReturn => {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Use ref to track if we're currently processing a stream to prevent race conditions
  const isProcessingRef = useRef(false);
  const abortControllerRef = useRef<AbortController | null>(null);

  // Memoize the expensive session events processing
  const processedEvents = useMemo((): EventItem[] => {
    if (!session) {
      return [];
    }

    const sessionEvents: EventItem[] = [];

    session.messages.forEach((msg: AgentMessage) => {
      if (msg.message.role === "assistant") {
        if (msg.message.content) {
          sessionEvents.push({
            type: "message",
            content: msg.message.content,
            metadata: {
              role: msg.message.role,
              agent: msg.agentName,
            },
          });
        }
        if (msg.message.tool_calls) {
          msg.message.tool_calls.forEach((toolCall) => {
            sessionEvents.push({
              type: "tool_call",
              content: "",
              metadata: {
                toolName: toolCall.function.name,
                toolArgs: toolCall.function.arguments,
                toolId: toolCall.id,
              },
            });
          });
        }
      } else if (msg.message.role === "user") {
        sessionEvents.push({
          type: "message",
          content: msg.message.content,
          metadata: {
            role: msg.message.role,
            agent: msg.agentName,
          },
        });
      } else if (msg.message.role === "tool") {
        sessionEvents.push({
          type: "tool_result",
          content: msg.message.content,
          metadata: {
            toolId: msg.message.tool_call_id,
            agent: msg.agentName,
          },
        });
      }
    });

    return sessionEvents;
  }, [session]);

  // Update events when processed events change
  useEffect(() => {
    setEvents(processedEvents);
    setError(null); // Clear errors when switching sessions
  }, [processedEvents]);

  // Memoize the event parsing logic
  const parseEventData = useCallback((line: string): Event | null => {
    if (!line.startsWith("data: ")) {
      return null;
    }

    try {
      return JSON.parse(line.slice(6)) as Event;
    } catch (e) {
      console.error("Failed to parse event:", e);
      return null;
    }
  }, []);

  // Memoize the stream processing logic
  const processStreamEvent = useCallback((eventData: Event) => {
    if (eventData.choice?.delta.content) {
      setEvents((prev) => {
        const lastEvent = prev[prev.length - 1];
        if (
          lastEvent?.type === "message" &&
          lastEvent.metadata?.role === eventData.choice?.delta?.role
        ) {
          return [
            ...prev.slice(0, -1),
            {
              ...lastEvent,
              content: lastEvent.content + eventData.choice!.delta.content,
            },
          ];
        }
        return [
          ...prev,
          {
            type: "message",
            content: eventData.choice!.delta.content,
            metadata: {
              agent: eventData.agent,
              role: eventData.choice?.delta?.role,
            },
          },
        ];
      });
    } else if (eventData.tool_call?.function && !eventData.response) {
      const {
        id,
        function: { name, arguments: args },
      } = eventData.tool_call;
      setEvents((prev) => [
        ...prev,
        {
          type: "tool_call",
          content: "",
          metadata: {
            toolName: name,
            toolArgs: args,
            toolId: id,
          },
        },
      ]);
    } else if (eventData.response) {
      setEvents((prev) => {
        const lastToolCall = [...prev]
          .reverse()
          .find((e) => e.type === "tool_call");
        return [
          ...prev,
          {
            type: "tool_result",
            content: eventData.response || "",
            metadata: {
              toolId: lastToolCall?.metadata?.toolName,
              agent: lastToolCall?.metadata?.agent,
            },
          },
        ];
      });
    } else if (eventData.message?.content) {
      setEvents((prev) => [
        ...prev,
        {
          type: "message",
          content: eventData.message!.content,
          metadata: {
            role: eventData.message!.role,
            agent: eventData.agent,
          },
        },
      ]);
    } else if (eventData.error?.message) {
      setEvents((prev) => [
        ...prev,
        {
          type: "error",
          content: eventData.error!.message,
        },
      ]);
    }
  }, []);

  const handleSubmit = useCallback(
    async (sessionId: string, prompt: string): Promise<void> => {
      if (!sessionId || !selectedAgent || isProcessingRef.current) {
        return;
      }

      // Cancel any existing request
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }

      // Create new abort controller for this request
      abortControllerRef.current = new AbortController();
      isProcessingRef.current = true;

      setIsLoading(true);
      setError(null);

      // Add user message to events
      setEvents((prev) => [
        ...prev,
        {
          type: "message",
          content: prompt,
          metadata: {
            role: "user",
          },
        },
      ]);

      try {
        const response = await fetch(
          `/api/sessions/${sessionId}/agent/${selectedAgent}`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify([
              {
                role: "user",
                content: prompt,
              },
            ]),
            signal: abortControllerRef.current.signal,
          }
        );

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error("No reader available - streaming not supported");
        }

        const decoder = new TextDecoder();

        try {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const text = decoder.decode(value, { stream: true });
            const lines = text.split("\n").filter((line) => line.trim());

            lines.forEach((line) => {
              const eventData = parseEventData(line);
              if (eventData) {
                processStreamEvent(eventData);
              }
            });
          }
        } finally {
          reader.releaseLock();
        }

        // Refresh sessions after streaming completes to update session data
        if (refreshSessions) {
          try {
            await refreshSessions();
          } catch (error) {
            console.warn(
              "Failed to refresh sessions after message completion:",
              error
            );
          }
        }
      } catch (error) {
        if (error instanceof Error && error.name === "AbortError") {
          // Request was cancelled, don't show error
          return;
        }

        const errorMessage =
          error instanceof Error ? error.message : "An unknown error occurred";
        console.error("Error:", error);
        setError(errorMessage);
        setEvents((prev) => [
          ...prev,
          {
            type: "error",
            content: errorMessage,
          },
        ]);
      } finally {
        setIsLoading(false);
        isProcessingRef.current = false;
        abortControllerRef.current = null;
      }
    },
    [selectedAgent, parseEventData, processStreamEvent, refreshSessions]
  );

  // Cleanup effect to cancel ongoing requests
  useEffect(() => {
    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort();
      }
      isProcessingRef.current = false;
    };
  }, []);

  // Memoize the return object to prevent unnecessary re-renders
  const returnValue = useMemo(
    (): UseEventsReturn => ({
      events,
      isLoading,
      error,
      handleSubmit,
    }),
    [events, isLoading, error, handleSubmit]
  );

  return returnValue;
};
