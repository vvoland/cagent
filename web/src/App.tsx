import { useState } from "react";
import "./App.css";
import { Sidebar } from "./components/Sidebar";
import { ToolCallEvent, ToolResultEvent } from "./components/ToolEvents";
import {
  MessageEvent,
  ErrorEvent,
  ChoiceEvents,
} from "./components/MessageEvents";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import type { EventItem } from "./types";

function App() {
  const [prompt, setPrompt] = useState("");
  const { sessions, currentSessionId, createNewSession, selectSession } =
    useSessions();
  const { events, isLoading, handleSubmit } = useEvents(
    currentSessionId,
    sessions
  );

  const handleFormSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentSessionId) return;
    await handleSubmit(currentSessionId, prompt);
    setPrompt("");
  };

  // Group consecutive choice events together
  const groupedEvents = events.reduce<(EventItem | EventItem[])[]>(
    (acc, event) => {
      if (event.type === "choice") {
        const lastGroup = acc[acc.length - 1];
        if (Array.isArray(lastGroup) && lastGroup[0].type === "choice") {
          lastGroup.push(event);
        } else {
          acc.push([event]);
        }
      } else {
        acc.push(event);
      }
      return acc;
    },
    []
  );

  return (
    <div className="app-container">
      <Sidebar
        sessions={sessions}
        currentSessionId={currentSessionId}
        onSessionSelect={selectSession}
      />
      <div className="main-container">
        <div className="header">
          <button onClick={createNewSession} className="new-session-button">
            New Session
          </button>
        </div>
        <div className="container">
          <div className="response">
            {groupedEvents.map((event, index) => {
              if (Array.isArray(event)) {
                return <ChoiceEvents key={index} events={event} />;
              }

              switch (event.type) {
                case "tool_call":
                  return (
                    <ToolCallEvent
                      key={index}
                      name={event.metadata?.toolName || ""}
                      args={event.metadata?.toolArgs || ""}
                    />
                  );
                case "tool_result":
                  return (
                    <ToolResultEvent
                      key={index}
                      id={event.metadata?.toolId || ""}
                      content={event.content}
                    />
                  );
                case "message":
                  return (
                    <MessageEvent
                      key={index}
                      role={event.metadata?.role || ""}
                      content={event.content}
                    />
                  );
                case "error":
                  return <ErrorEvent key={index} content={event.content} />;
                default:
                  return null;
              }
            })}
          </div>

          <form onSubmit={handleFormSubmit} className="form">
            <input
              type="text"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Enter your prompt..."
              disabled={isLoading || !currentSessionId}
              className="input"
            />
            <button
              type="submit"
              disabled={isLoading || !currentSessionId}
              className="button"
            >
              {isLoading ? "Processing..." : "Submit"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default App;
