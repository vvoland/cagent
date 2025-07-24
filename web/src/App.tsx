import { useState, useEffect, useCallback, memo, Suspense, useRef, useMemo } from "react";
import { useSessions } from "./hooks/useSessions";
import { useEvents } from "./hooks/useEvents";
import { useAgents } from "./hooks/useAgents";
import { useKeyboard, commonShortcuts } from "./hooks/useKeyboard";
import { useLogger } from "./utils/logger";
import { useToastHelpers } from "./components/Toast";
import { Button } from "./components/ui/button";
import { Input } from "./components/ui/input";
import { MessageEvent, ErrorEvent } from "./components/MessageEvents";
import { StackedToolEvents } from "./components/ToolEvents";
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
  const [pendingPrompt, setPendingPrompt] = useState<string | null>(null);
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
    refreshSessions,
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
  } = useEvents(currentSessionId, sessions, selectedAgent, refreshSessions);

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

  // Handle pending prompt submission if session is created
  useEffect(() => {
  if (currentSessionId && pendingPrompt) {
    // Session was created and we have a pending prompt to submit
    const submitPendingPrompt = async () => {
      try {
        await handleSubmit(currentSessionId, pendingPrompt);
        setPrompt("");
        logger.info('Pending prompt submitted successfully', { prompt: pendingPrompt.slice(0, 50) + '...' });
        toast.success('Message sent successfully');
      } catch (error) {
        logger.error('Failed to submit pending prompt', error);
        toast.error('Failed to send message', 'Please try again');
      } finally {
        setPendingPrompt(null); // Clear the pending prompt
      }
    };
    
    submitPendingPrompt();
  }
}, [currentSessionId, pendingPrompt, handleSubmit, logger, toast]);

  const handleFormSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedAgent || !prompt.trim()) return;

    logger.time('submit-prompt');
    try {

      let sessionId = currentSessionId;

      // Auto-create session if none exists
      if (!sessionId) {
        logger.info('No session exists, creating new session automatically for replay');
        setPendingPrompt(prompt); // Store the prompt to submit after session creation
        await handleNewSession();
        return;
      }

      await handleSubmit(sessionId, prompt);
      setPrompt("");
      logger.info('Prompt submitted successfully', { prompt: prompt.slice(0, 50) + '...' });
      toast.success('Message sent successfully');
    } catch (error) {
      logger.error('Failed to submit prompt', error);
      toast.error('Failed to send message', 'Please try again');
    } finally {
      logger.timeEnd('submit-prompt');
    }
  }, [currentSessionId, createNewSession, handleSubmit, prompt, selectedAgent, logger, toast]);

  // Replay message handler
  const handleReplayMessage = useCallback(async (content: string) => {
    if (!selectedAgent) return;

    logger.info('Replaying message', { content: content.slice(0, 50) + '...' });
    try {
      let sessionId = currentSessionId;

      // Auto-create session if none exists
      if (!sessionId) {
        logger.info('No session exists, error');
        sessionId = await createNewSession();
        if (!sessionId) {
          toast.error('Failed to create session for replay', 'Please try again');
          throw new Error('Failed to create session');
        }
        toast.info('New session created for replay');
      }

      await handleSubmit(sessionId, content);
      toast.success('Message replayed successfully');
    } catch (error) {
      logger.error('Failed to replay message', error);
      toast.error('Failed to replay message', 'Please try again');
      throw error; // Re-throw to handle loading state in MessageActionButtons
    }
  }, [currentSessionId, createNewSession, handleSubmit, selectedAgent, logger, toast]);

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

  // Process events into chronological order with tool call stacks
  const processedEvents = useMemo(() => {
    const processed: Array<{
      type: 'message' | 'error' | 'tool_stack';
      data: any;
      key: string;
    }> = [];

    let currentToolStack: Array<{
      id: string;
      callEvent: {
        name: string;
        args: string;
        timestamp: Date;
      };
      resultEvent?: {
        id: string;
        content: string;
        success: boolean;
        timestamp: Date;
      };
      status: 'pending' | 'success' | 'error';
    }> = [];

    let toolStackId = 0;

    events.forEach((event, index) => {
      if (event.type === 'tool_call') {
        // Add to current tool stack
        const toolId = event.metadata?.toolId || `tool-${Date.now()}-${Math.random()}`;
        currentToolStack.push({
          id: toolId,
          callEvent: {
            name: event.metadata?.toolName || 'unknown',
            args: event.metadata?.toolArgs || '{}',
            timestamp: event.timestamp || new Date()
          },
          status: 'pending'
        });
      } else if (event.type === 'tool_result') {
        // Find the corresponding tool call in current stack and update it
        const toolIndex = currentToolStack.findIndex(tool =>
          !tool.resultEvent && tool.status === 'pending'
        );

        if (toolIndex !== -1) {
          const existingTool = currentToolStack[toolIndex];
          if (existingTool) {
            currentToolStack[toolIndex] = {
              id: existingTool.id,
              callEvent: existingTool.callEvent,
              resultEvent: {
                id: event.metadata?.toolId || 'unknown',
                content: event.content || '',
                success: event.metadata?.success !== false,
                timestamp: event.timestamp || new Date()
              },
              status: event.metadata?.success !== false ? 'success' : 'error'
            };
          }
        }
      } else {
        // Non-tool event (message, error)
        // First, flush any pending tool stack
        if (currentToolStack.length > 0) {
          processed.push({
            type: 'tool_stack',
            data: [...currentToolStack],
            key: `tool-stack-${toolStackId++}`
          });
          currentToolStack = [];
        }

        // Add the message/error event
        processed.push({
          type: event.type as 'message' | 'error',
          data: event,
          key: `${event.type}-${index}`
        });
      }
    });

    // Flush any remaining tool stack at the end
    if (currentToolStack.length > 0) {
      processed.push({
        type: 'tool_stack',
        data: [...currentToolStack],
        key: `tool-stack-${toolStackId++}`
      });
    }

    return processed;
  }, [events]);

  // Render processed events in chronological order
  const renderProcessedEvent = useCallback((processedEvent: any, index: number) => {
    const { type, data, key } = processedEvent;

    switch (type) {
      case 'tool_stack':
        return (
          <div key={key} className="mx-2 lg:mx-3">
            <StackedToolEvents
              events={data.map((tool: any) => ([
                {
                  type: 'tool_call' as const,
                  name: tool.callEvent.name,
                  args: tool.callEvent.args,
                  id: tool.id,
                  timestamp: tool.callEvent.timestamp
                },
                ...(tool.resultEvent ? [{
                  type: 'tool_result' as const,
                  id: tool.resultEvent.id,
                  content: tool.resultEvent.content,
                  success: tool.resultEvent.success,
                  timestamp: tool.resultEvent.timestamp
                }] : [])
              ])).flat()}
            />
          </div>
        );
      case 'message':
        const isLatestEvent = index === processedEvents.length - 1;
        return (
          <MessageEvent
            key={key}
            role={data.metadata?.role || ""}
            agent={data.metadata?.agent || ""}
            content={data.content}
            onReplay={data.metadata?.role === 'user' ? (() => handleReplayMessage(data.content)) : undefined}
            isLatest={isLatestEvent}
          />
        );
      case 'error':
        return <ErrorEvent key={key} content={data.content} />;
      default:
        return null;
    }
  }, [handleReplayMessage, processedEvents.length]);

  // Check if form is disabled (allow submission without session - will auto-create)
  const isFormDisabled = isLoadingEvents || !selectedAgent;

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
            <div className="flex-1 overflow-y-auto min-h-0">
              <Sidebar
                sessions={sessions}
                currentSessionId={currentSessionId}
                onSessionSelect={handleSessionSelect}
                onDeleteSession={handleDeleteSession}
                isMobile={true}
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
              {/* Render all events in chronological order with tool stacks */}
              {processedEvents.map((processedEvent, index) =>
                renderProcessedEvent(processedEvent, index)
              )}
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