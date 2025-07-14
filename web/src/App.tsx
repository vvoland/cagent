import { useState, useEffect } from "react";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import { useAgents } from "./hooks/useAgents";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { MessageEvent, ErrorEvent } from "./components/MessageEvents";
import { ToolCallEvent, ToolResultEvent } from "./components/ToolEvents";
import { Sidebar } from "./components/Sidebar";
import { DarkModeToggle } from "./components/DarkModeToggle";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
function App() {
  const [prompt, setPrompt] = useState("");
  const {
    sessions,
    currentSessionId,
    createNewSession,
    selectSession,
    deleteSession,
  } = useSessions();
  const { agents, selectedAgent, setSelectedAgent } = useAgents();

  // Update selected agent when session changes
  useEffect(() => {
    if (currentSessionId && sessions.find((s) => s.id === currentSessionId)) {
      const session = sessions.find((s) => s.id === currentSessionId);
      // Get the agent name from the first message
      if (session && session.messages && session.messages.length > 0) {
        const agentName = session.messages[0].agentName;
        setSelectedAgent(agentName);
      }
    }
  }, [currentSessionId, sessions, setSelectedAgent]);

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

  return (
    <div className="min-h-screen flex bg-gray-200 dark:bg-background text-black dark:text-white">
      <Sidebar
        sessions={sessions}
        currentSessionId={currentSessionId}
        onSessionSelect={selectSession}
        onDeleteSession={deleteSession}
      />
      <div className="flex-1 flex flex-col h-screen">
        <div className="p-4 border-b dark:border-border">
          <div className="flex gap-4 items-center justify-between">
            <div className="flex gap-4 items-center">
              <Button onClick={() => createNewSession()} variant="outline">
                New Session
              </Button>

              <Select
                value={selectedAgent || ""}
                onValueChange={(value: string) => setSelectedAgent(value)}
              >
                <SelectTrigger className="w-[380px]">
                  <SelectValue placeholder="Select an agent..." />
                </SelectTrigger>
                <SelectContent>
                  {agents.map((agent) => (
                    <SelectItem key={agent.name} value={agent.name}>
                      {agent.name} - {agent.description}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <DarkModeToggle />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-4 pb-24">
          <div className="max-w-4xl mx-auto space-y-4">
            {events.map((event, index) => {
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
                      agent={event.metadata?.agent || ""}
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

        <div className="border-t dark:border-border bg-white dark:bg-background shadow-lg">
          <div className="max-w-4xl mx-auto p-4">
            <form onSubmit={handleFormSubmit} className="flex gap-2">
              <Input
                value={prompt}
                onChange={(e) => setPrompt(e.target.value)}
                placeholder="Enter your prompt..."
                disabled={
                  isLoadingEvents || !currentSessionId || !selectedAgent
                }
                className="flex-1"
              />
              <Button
                type="submit"
                disabled={
                  isLoadingEvents || !currentSessionId || !selectedAgent
                }
              >
                {isLoadingEvents ? "Processing..." : "Submit"}
              </Button>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}

export default App;
