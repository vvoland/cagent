import { useState, useEffect, useCallback, memo, Suspense, useRef } from "react";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import { useAgents } from "./hooks/useAgents";
import { useKeyboard, commonShortcuts } from "./hooks/useKeyboard";
import { useLogger } from "./utils/logger";
import { useToastHelpers } from "./components/Toast";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { MessageEvent, ErrorEvent } from "./components/MessageEvents";
import { ToolCallEvent, ToolResultEvent, ConnectedToolEvents } from "./components/ToolEvents";
import { Sidebar } from "./components/Sidebar";
import { DarkModeToggle } from "./components/DarkModeToggle";
import { SkeletonList, MessageSkeleton } from "./components/LoadingSkeleton";
import { Menu, X, ChevronDown } from "lucide-react";
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
  const [isNearBottom, setIsNearBottom] = useState(true);
  const [showScrollButton, setShowScrollButton] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const logger = useLogger('App');
  const toast = useToastHelpers();
  
  // Refs for scroll management
  const scrollContainerRef = useRef<HTMLElement>(null);
  const lastEventCountRef = useRef(0);
  
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

  // Scroll detection logic
  const checkScrollPosition = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const { scrollTop, scrollHeight, clientHeight } = container;
    const distanceFromBottom = scrollHeight - scrollTop - clientHeight;
    const nearBottom = distanceFromBottom <= 100; // 100px threshold
    
    setIsNearBottom(nearBottom);
    setShowScrollButton(!nearBottom && events.length > 0);
  }, [events.length]);

  // Auto-scroll when new messages arrive (only if user was near bottom)
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const currentEventCount = events.length;
    const hasNewEvents = currentEventCount > lastEventCountRef.current;
    
    if (hasNewEvents) {
      if (isNearBottom) {
        // Auto-scroll to bottom with smooth animation
        container.scrollTo({
          top: container.scrollHeight,
          behavior: 'smooth'
        });
        setUnreadCount(0); // Reset unread count when auto-scrolling
      } else {
        // User has scrolled up, increment unread count
        const newMessages = currentEventCount - lastEventCountRef.current;
        setUnreadCount(prev => prev + newMessages);
      }
    }
    
    lastEventCountRef.current = currentEventCount;
  }, [events.length, isNearBottom]);

  // Scroll to bottom handler
  const scrollToBottom = useCallback(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    container.scrollTo({
      top: container.scrollHeight,
      behavior: 'smooth'
    });
    setUnreadCount(0);
    setShowScrollButton(false);
  }, []);

  // Attach scroll event listener
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const handleScroll = () => {
      requestAnimationFrame(checkScrollPosition);
    };

    container.addEventListener('scroll', handleScroll, { passive: true });
    
    // Initial check
    checkScrollPosition();

    return () => {
      container.removeEventListener('scroll', handleScroll);
    };
  }, [checkScrollPosition]);

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

  // Replay message handler
  const handleReplayMessage = useCallback(async (content: string) => {
    if (!currentSessionId) return;
    
    logger.info('Replaying message', { content: content.slice(0, 50) + '...' });
    try {
      await handleSubmit(currentSessionId, content);
      toast.success('Message replayed successfully');
    } catch (error) {
      logger.error('Failed to replay message', error);
      toast.error('Failed to replay message', 'Please try again');
      throw error; // Re-throw to handle loading state in MessageActionButtons
    }
  }, [currentSessionId, handleSubmit, logger, toast]);

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

  // Helper function to group tool call/result pairs
  const groupToolEvents = useCallback((events: any[]) => {
    const grouped: any[] = [];
    let i = 0;
    
    while (i < events.length) {
      const event = events[i];
      
      if (event.type === 'tool_call') {
        // Look for a matching tool_result in the next few events
        const toolEvents: any[] = [{
          type: 'tool_call',
          name: event.metadata?.toolName || '',
          args: event.metadata?.toolArgs || '{}',
          timestamp: event.timestamp
        }];
        
        // Check if the next event is a tool_result
        if (i + 1 < events.length && events[i + 1].type === 'tool_result') {
          const resultEvent = events[i + 1];
          toolEvents.push({
            type: 'tool_result',
            id: resultEvent.metadata?.toolId || '',
            content: resultEvent.content || '',
            success: !resultEvent.metadata?.error,
            timestamp: resultEvent.timestamp
          });
          i += 2; // Skip both events
        } else {
          i += 1; // Only skip the tool_call event
        }
        
        grouped.push({
          type: 'connected_tools',
          events: toolEvents,
          index: i
        });
      } else {
        // Non-tool events remain as-is
        grouped.push({ ...event, index: i });
        i += 1;
      }
    }
    
    return grouped;
  }, []);

  // Memoized render function for events
  const renderEvent = useCallback((eventGroup: any, index: number) => {
    if (eventGroup.type === 'connected_tools') {
      return (
        <ConnectedToolEvents
          key={`connected-tools-${index}`}
          events={eventGroup.events}
          className="mx-2 lg:mx-3"
        />
      );
    }
    
    // Handle regular events
    const event = eventGroup;
    switch (event.type) {
      case "tool_call":
        // Fallback for individual tool calls (shouldn't happen with grouping)
        return (
          <ToolCallEvent
            key={`${event.type}-${index}`}
            name={event.metadata?.toolName || ""}
            args={event.metadata?.toolArgs || ""}
          />
        );
      case "tool_result":
        // Fallback for individual tool results (shouldn't happen with grouping)
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
            onReplay={event.metadata?.role === 'user' ? (() => handleReplayMessage(event.content)) : undefined}
          />
        );
      case "error":
        return <ErrorEvent key={`${event.type}-${index}`} content={event.content} />;
      default:
        return null;
    }
  }, [handleReplayMessage]);

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
        <main 
          ref={scrollContainerRef}
          className="flex-1 overflow-y-auto p-3 lg:p-4 pb-20 lg:pb-24 relative"
        >
          <div className="max-w-4xl mx-auto space-y-4">
            <Suspense fallback={
              <SkeletonList count={3} component={MessageSkeleton} />
            }>
              {groupToolEvents(events).map(renderEvent)}
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
          
          {/* Scroll to bottom button */}
          {showScrollButton && (
            <button
              onClick={scrollToBottom}
              className={`
                fixed bottom-24 lg:bottom-28 right-4 lg:right-6 z-30
                w-12 h-12 lg:w-10 lg:h-10 rounded-full
                bg-blue-500 hover:bg-blue-600 dark:bg-[#5e81ac] dark:hover:bg-[#81a1c1]
                text-white shadow-xl hover:shadow-2xl
                border border-blue-600 dark:border-[#4c566a]
                flex items-center justify-center
                transition-all duration-300 ease-in-out
                hover:-translate-y-1 hover:scale-110
                focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-2
                dark:focus:ring-[#81a1c1] dark:focus:ring-offset-gray-900
                group
              `}
              aria-label={`Scroll to bottom${unreadCount > 0 ? ` (${unreadCount} new messages)` : ''}`}
              title={`Scroll to bottom${unreadCount > 0 ? ` (${unreadCount} new messages)` : ''}`}
            >
              <div className="relative flex items-center justify-center">
                <ChevronDown className="w-5 h-5 lg:w-4 lg:h-4 group-hover:animate-bounce" />
                {unreadCount > 0 && (
                  <div className="absolute -top-2 -right-2 w-5 h-5 lg:w-4 lg:h-4 bg-red-500 dark:bg-[#bf616a] rounded-full flex items-center justify-center border-2 border-white dark:border-gray-900">
                    <span className="text-xs font-bold text-white leading-none">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </span>
                  </div>
                )}
              </div>
            </button>
          )}
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