import { useState, useEffect } from "react";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import { useAgents } from "./hooks/useAgents";
import type { EventItem } from "./types";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import {
  MessageEvent,
  ErrorEvent,
  ChoiceEvents,
} from "./components/MessageEvents";
import { ToolCallEvent, ToolResultEvent } from "./components/ToolEvents";
import { Sidebar } from "./components/Sidebar";

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
    <div className="min-h-screen flex bg-background">
      <Sidebar
        sessions={sessions}
        currentSessionId={currentSessionId}
        onSessionSelect={selectSession}
      />
      <div className="flex-1 flex flex-col">
        <div className="p-4 border-b">
          <div className="flex gap-4 items-center">
            <Button
              onClick={() => selectedAgent && createNewSession(selectedAgent)}
              disabled={!selectedAgent}
              variant="outline"
            >
              New Session
            </Button>
            <select
              value={selectedAgent || ""}
              onChange={(e) => setSelectedAgent(e.target.value)}
              disabled={isLoadingAgents}
              className="flex h-9 w-[200px] rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus:outline-none focus:ring-1 focus:ring-ring"
            >
              <option value="">Select an agent...</option>
              {agents.map((agent) => (
                <option key={agent.name} value={agent.name}>
                  {agent.name} - {agent.description}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="flex-1 overflow-auto p-4">
          <div className="space-y-4">
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
        </div>

        <div className="p-4 border-t">
          <form onSubmit={handleFormSubmit} className="flex gap-2">
            <Input
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              placeholder="Enter your prompt..."
              disabled={isLoadingEvents || !currentSessionId || !selectedAgent}
              className="flex-1"
            />
            <Button
              type="submit"
              disabled={isLoadingEvents || !currentSessionId || !selectedAgent}
            >
              {isLoadingEvents ? "Processing..." : "Submit"}
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}

export default App;
