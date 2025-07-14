import { useState, useEffect } from "react";
import type { EventItem, Event, AgentMessage, Session } from "../types";

export const useEvents = (
  sessionId: string | null,
  sessions: Session[],
  selectedAgent: string | null
) => {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    if (sessionId && sessions.find((s) => s.id === sessionId)) {
      const session = sessions.find((s) => s.id === sessionId);
      const sessionEvents: EventItem[] = [];

      if (session && Array.isArray(session.messages)) {
        session.messages.forEach((msg: AgentMessage) => {
          if (msg.message.role === "assistant") {
            if (msg.message.content) {
              sessionEvents.push({
                type: "message",
                content: msg.message.content,
                metadata: {
                  role: msg.message.role,
                  agent: msg.agent.name,
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
                agent: msg.agent.name,
              },
            });
          } else if (msg.message.role === "tool") {
            sessionEvents.push({
              type: "tool_result",
              content: msg.message.content,
              metadata: {
                toolId: msg.message.tool_call_id,
                agent: msg.agent.name,
              },
            });
          }
        });
      }

      setEvents(sessionEvents);
    } else {
      setEvents([]);
    }
  }, [sessionId, sessions]);

  const handleSubmit = async (sessionId: string, prompt: string) => {
    if (!sessionId || !selectedAgent) return;
    setIsLoading(true);
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
        }
      );

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error("No reader available");
      }

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const text = new TextDecoder().decode(value);
        const lines = text.split("\n").filter((line) => line.trim());

        lines.forEach((line) => {
          if (line.startsWith("data: ")) {
            try {
              const eventData = JSON.parse(line.slice(6)) as Event;

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
                        content:
                          lastEvent.content + eventData.choice!.delta.content,
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
            } catch (e) {
              console.error("Failed to parse event:", e);
            }
          }
        });
      }
    } catch (error) {
      console.error("Error:", error);
      setEvents((prev) => [
        ...prev,
        {
          type: "error",
          content: (error as Error).message,
        },
      ]);
    } finally {
      setIsLoading(false);
    }
  };

  return { events, isLoading, handleSubmit };
};
