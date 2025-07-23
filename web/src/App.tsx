import { useState, useEffect, useCallback, memo, Suspense } from "react";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import { useAgents } from "./hooks/useAgents";
import { useKeyboard, commonShortcuts } from "./hooks/useKeyboard";
import { useLogger } from "./utils/logger";
import { useToastHelpers } from "./components/Toast";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { MessageEvent, ErrorEvent } from "./components/MessageEvents";
import { ToolCallEvent, ToolResultEvent } from "./components/ToolEvents";
import { Sidebar } from "./components/Sidebar";
import { DarkModeToggle } from "./components/DarkModeToggle";
import { SkeletonList, MessageSkeleton } from "./components/LoadingSkeleton";
import { Menu, X } from "lucide-react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const App = memo(() => {
  const [prompt, setPrompt] = useState("");
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const logger = useLogger('App');
  const toast = useToastHelpers();
  
  const {
    sessions,
    currentSessionId,
    createNewSession,
    selectSession,
    deleteSession,
  } = useSessions();
  const { agents, selectedAgent, setSelectedAgent } = useAgents();

  // Memoized callbacks for better performance
  const handleNewSession = useCallback(() => {
    logger.info('Creating new session');
    createNewSession();
    toast.success('New session created');
    setIsMobileMenuOpen(false); // Close mobile menu after action
  }, [createNewSession, logger, toast]);

  const handleSessionSelect = useCallback((sessionId: string) => {
    logger.info('Selecting session', { sessionId });
    selectSession(sessionId);
    setIsMobileMenuOpen(false); // Close mobile menu after selection
  }, [selectSession, logger]);

  const handleDeleteSession = useCallback((sessionId: string) => {
    logger.info('Deleting session', { sessionId });
    deleteSession(sessionId);
    toast.info('Session deleted');
  }, [deleteSession, logger, toast]);

  const handleAgentChange = useCallback((value: string) => {
    logger.info('Changing agent', { agent: value });
    setSelectedAgent(value);
    toast.info(`Agent changed to ${value}`);
  }, [setSelectedAgent, logger, toast]);

  const handlePromptChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    setPrompt(e.target.value);
  }, []);

  const toggleMobileMenu = useCallback(() => {
    setIsMobileMenuOpen(prev => !prev);
  }, []);

  // Close mobile menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      const target = event.target as Element;
      if (isMobileMenuOpen && !target.closest('.mobile-sidebar') && !target.closest('.mobile-menu-button')) {
        setIsMobileMenuOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, [isMobileMenuOpen]);

  // Update selected agent when session changes
  useEffect(() => {
    if (currentSessionId && sessions.find((s) => s.id === currentSessionId)) {
      const session = sessions.find((s) => s.id === currentSessionId);
      // Get the agent name from the first message
      if (session && session.messages && session.messages.length > 0) {
        const agentName = session.messages[0]?.agentName;
        if (agentName) {
          setSelectedAgent(agentName);
        }
      }
    }
  }, [currentSessionId, sessions, setSelectedAgent]);

  const {
    events,
    isLoading: isLoadingEvents,
    handleSubmit,
  } = useEvents(currentSessionId, sessions, selectedAgent);

  const handleFormSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentSessionId) return;
    
    logger.time('submit-prompt');
    try {
      await handleSubmit(currentSessionId, prompt);
      setPrompt("");
      logger.info('Prompt submitted successfully', { prompt: prompt.slice(0, 50) + '...' });
      toast.success('Message sent successfully');
    } catch (error) {
      logger.error('Failed to submit prompt', error);
      toast.error('Failed to send message', 'Please try again');
    } finally {
      logger.timeEnd('submit-prompt');
    }
  }, [currentSessionId, handleSubmit, prompt, logger, toast]);

  // Keyboard shortcuts (disabled on mobile)
  useKeyboard([
    {
      ...commonShortcuts.newSession,
      handler: handleNewSession,
    },
    {
      ...commonShortcuts.toggleTheme,
      handler: () => {
        // This would be handled by the DarkModeToggle component
        const event = new KeyboardEvent('keydown', { 
          key: 'd', 
          ctrlKey: true 
        });
        document.dispatchEvent(event);
      },
    },
  ], { enabled: window.innerWidth > 768 }); // Disable keyboard shortcuts on mobile

  // Performance logging
  useEffect(() => {
    logger.info('App component mounted');
    return () => {
      logger.info('App component unmounted');
    };
  }, [logger]);

  // Memoized render function for events
  const renderEvent = useCallback((event: any, index: number) => {
    switch (event.type) {
      case "tool_call":
        return (
          <ToolCallEvent
            key={`${event.type}-${index}`}
            name={event.metadata?.toolName || ""}
            args={event.metadata?.toolArgs || ""}
          />
        );
      case "tool_result":
        return (
          <ToolResultEvent
            key={`${event.type}-${index}`}
            id={event.metadata?.toolId || ""}
            content={event.content}
          />
        );
      case "message":
        return (
          <MessageEvent
            key={`${event.type}-${index}`}
            role={event.metadata?.role || ""}
            agent={event.metadata?.agent || ""}
            content={event.content}
          />
        );
      case "error":
        return <ErrorEvent key={`${event.type}-${index}`} content={event.content} />;
      default:
        return null;
    }
  }, []);

  // Check if form is disabled
  const isFormDisabled = isLoadingEvents || !currentSessionId || !selectedAgent;

  return (
    <div className="min-h-screen flex bg-gray-200 dark:bg-background text-black dark:text-white">
      {/* Desktop Sidebar */}
      <div className="hidden lg:block">
        <Sidebar
          sessions={sessions}
          currentSessionId={currentSessionId}
          onSessionSelect={handleSessionSelect}
          onDeleteSession={handleDeleteSession}
        />
      </div>

      {/* Mobile Sidebar Overlay */}
      {isMobileMenuOpen && (
        <div className="lg:hidden fixed inset-0 bg-black/50 z-40 backdrop-blur-sm">
          <div className="mobile-sidebar fixed left-0 top-0 h-full w-80 max-w-[85vw] bg-white dark:bg-background border-r dark:border-border shadow-xl transform transition-transform duration-300 ease-in-out">
            <div className="p-4 border-b dark:border-border flex items-center justify-between">
              <h2 className="text-lg font-semibold">Sessions</h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={toggleMobileMenu}
                className="h-8 w-8"
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <div className="flex-1 overflow-hidden">
              <Sidebar
                sessions={sessions}
                currentSessionId={currentSessionId}
                onSessionSelect={handleSessionSelect}
                onDeleteSession={handleDeleteSession}
              />
            </div>
          </div>
        </div>
      )}

      <div className="flex-1 flex flex-col h-screen min-w-0">
        {/* Header */}
        <header className="p-3 lg:p-4 border-b dark:border-border bg-white/80 dark:bg-background/80 backdrop-blur-sm sticky top-0 z-30">
          <div className="flex gap-2 lg:gap-4 items-center justify-between">
            <div className="flex gap-2 lg:gap-4 items-center min-w-0 flex-1">
              {/* Mobile Menu Button */}
              <Button
                variant="outline"
                size="icon"
                onClick={toggleMobileMenu}
                className="mobile-menu-button lg:hidden h-9 w-9 flex-shrink-0"
              >
                <Menu className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>

              {/* New Session Button */}
              <Button 
                onClick={handleNewSession} 
                variant="outline"
                size="sm"
                className="transition-all hover:scale-105 hidden sm:flex"
              >
                New Session
              </Button>
              
              {/* Mobile New Session Button */}
              <Button 
                onClick={handleNewSession} 
                variant="outline"
                size="icon"
                className="sm:hidden h-9 w-9 flex-shrink-0"
                title="New Session"
              >
                <span className="text-lg">+</span>
              </Button>

              {/* Agent Selector */}
              <div className="min-w-0 flex-1 max-w-sm lg:max-w-md">
                <Select
                  value={selectedAgent || ""}
                  onValueChange={handleAgentChange}
                >
                  <SelectTrigger className="w-full transition-all hover:shadow-md text-sm">
                    <SelectValue placeholder="Select agent..." />
                  </SelectTrigger>
                  <SelectContent>
                    {agents.map((agent) => (
                      <SelectItem key={agent.name} value={agent.name}>
                        <div className="flex flex-col sm:flex-row sm:items-center">
                          <span className="font-medium">{agent.name}</span>
                          <span className="text-muted-foreground text-xs sm:ml-2 sm:text-sm">
                            {agent.description}
                          </span>
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            
            {/* Dark Mode Toggle */}
            <DarkModeToggle />
          </div>
        </header>

        {/* Main content */}
        <main className="flex-1 overflow-y-auto p-3 lg:p-4 pb-20 lg:pb-24">
          <div className="max-w-4xl mx-auto space-y-4">
            <Suspense fallback={
              <SkeletonList count={3} component={MessageSkeleton} />
            }>
              {events.map(renderEvent)}
            </Suspense>
            
            {/* Loading indicator */}
            {isLoadingEvents && (
              <div className="flex items-center justify-center py-4">
                <div className="flex items-center gap-2 text-muted-foreground">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current"></div>
                  <span className="text-sm lg:text-base">Processing...</span>
                </div>
              </div>
            )}
            
            {/* Empty state */}
            {events.length === 0 && !isLoadingEvents && (
              <div className="flex flex-col items-center justify-center py-8 lg:py-12 text-center px-4">
                <div className="text-4xl lg:text-6xl mb-4 opacity-50">💬</div>
                <h3 className="text-base lg:text-lg font-semibold mb-2">Start a conversation</h3>
                <p className="text-muted-foreground text-sm lg:text-base max-w-md">
                  {selectedAgent 
                    ? `Send a message to ${selectedAgent} to get started.`
                    : 'Select an agent and send a message to begin.'
                  }
                </p>
              </div>
            )}
          </div>
        </main>

        {/* Footer form */}
        <footer className="border-t dark:border-border bg-white/95 dark:bg-background/95 backdrop-blur-sm shadow-lg sticky bottom-0 z-20">
          <div className="max-w-4xl mx-auto p-3 lg:p-4">
            <form onSubmit={handleFormSubmit} className="flex gap-2">
              <Input
                value={prompt}
                onChange={handlePromptChange}
                placeholder="Enter your message..."
                disabled={isFormDisabled}
                className="flex-1 transition-all focus:shadow-md text-base lg:text-sm min-h-[44px] lg:min-h-[36px]"
                autoComplete="off"
                maxLength={10000}
              />
              <Button
                type="submit"
                disabled={isFormDisabled}
                className="transition-all hover:scale-105 px-4 lg:px-6 min-h-[44px] lg:min-h-[36px]"
              >
                {isLoadingEvents ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2"></div>
                    <span className="hidden sm:inline">Processing...</span>
                    <span className="sm:hidden">...</span>
                  </>
                ) : (
                  <>
                    <span className="hidden sm:inline">Submit</span>
                    <span className="sm:hidden">Send</span>
                  </>
                )}
              </Button>
            </form>
          </div>
        </footer>
      </div>
    </div>
  );
});

App.displayName = 'App';

export default App;