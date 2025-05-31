import { useState, useEffect } from "react";
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
import { useAgents } from "./hooks/useAgents";
import type { EventItem } from "./types";

function App() {
  const [prompt, setPrompt] = useState("");
  const { sessions, currentSessionId, createNewSession, selectSession } =
    useSessions();
  const {
    agents,
    selectedAgent,
    setSelectedAgent,
    isLoading: isLoadingAgents,
  } = useAgents();

  // Update selected agent when session changes
  useEffect(() => {
    if (currentSessionId && sessions[currentSessionId]) {
      const session = sessions[currentSessionId];
      console.log("Session selected:", session);
      // Get the agent name from the first message
      if (session.messages && session.messages.length > 0) {
        const agentName = session.messages[0].agent.name;
        console.log("Setting selected agent to:", agentName);
        setSelectedAgent(agentName);
      }
    }
  }, [currentSessionId, sessions]);

  const {
    events,
    isLoading: isLoadingEvents,
    handleSubmit,
  } = useEvents(currentSessionId, sessions, selectedAgent);

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
          <button
            onClick={() => selectedAgent && createNewSession(selectedAgent)}
            className="new-session-button"
            disabled={!selectedAgent}
          >
            New Session
          </button>
          <select
            value={selectedAgent || ""}
            onChange={(e) => setSelectedAgent(e.target.value)}
            disabled={isLoadingAgents}
            className="agent-selector"
          >
            {agents.map((agent) => (
              <option key={agent.name} value={agent.name}>
                {agent.name} - {agent.description}
              </option>
            ))}
          </select>
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
              disabled={isLoadingEvents || !currentSessionId || !selectedAgent}
              className="input"
            />
            <button
              type="submit"
              disabled={isLoadingEvents || !currentSessionId || !selectedAgent}
              className="button"
            >
              {isLoadingEvents ? "Processing..." : "Submit"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default App;
