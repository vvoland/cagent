import { useState, useCallback, memo } from 'react';
import { cn } from '../lib/utils';
import { ChevronUp, ChevronDown } from 'lucide-react';
import { 
  ConnectedToolChip, 
  useConnectedToolCalls 
} from './ConnectedToolChip';

interface StackedToolEventsProps {
  events: Array<{
    type: 'tool_call' | 'tool_result';
    name?: string;
    args?: string;
    id?: string;
    content?: string;
    success?: boolean;
    timestamp?: Date;
  }>;
  className?: string;
  maxVisible?: number;
}

export const StackedToolEvents = memo<StackedToolEventsProps>(({ 
  events, 
  className, 
  maxVisible = 3
}) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const { connectedToolCalls, toggleExpanded, isExpanded: isChipExpanded } = useConnectedToolCalls(events);
  
  if (connectedToolCalls.length === 0) return null;

  // Show stacking effect when there are multiple tool calls
  const shouldShowStacking = connectedToolCalls.length > 1;
  const visibleTools = isExpanded ? connectedToolCalls : connectedToolCalls.slice(-maxVisible);
  const hiddenCount = Math.max(0, connectedToolCalls.length - maxVisible);
  
  const handleStackToggle = useCallback(() => {
    setIsExpanded(prev => !prev);
  }, []);

  const getStackStyle = (index: number, total: number) => {
      return {
        transform: 'scale(1)',
        opacity: 1,
        zIndex: 10 + index,
        clipPath: 'none',
        marginBottom: index < total - 1 ? '8px' : '0'
      };
  };

  return (
    <div className={cn("relative", className)}>
      {/* Stack Controls - Show expand button when there are hidden tools */}
      {shouldShowStacking && !isExpanded && hiddenCount > 0 && (
        <div className="mb-3 flex justify-center">
          <button
            onClick={handleStackToggle}
            className={cn(
              "inline-flex items-center gap-2 px-3 py-2 rounded-full bg-blue-100 hover:bg-blue-200 dark:bg-blue-900/50 dark:hover:bg-blue-800/50",
              "text-blue-800 dark:text-blue-200 border border-blue-300 dark:border-blue-700",
              "transition-all duration-200 hover:scale-105 active:scale-95 text-sm font-medium shadow-sm hover:shadow-md",
              "min-h-[44px] lg:min-h-[32px] focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-2",
              "dark:focus:ring-blue-400 dark:focus:ring-offset-gray-900"
            )}
            aria-label={`Expand to show ${hiddenCount} more tool calls`}
            aria-expanded={false}
          >
            <ChevronUp className="w-4 h-4" />
            <span className="whitespace-nowrap">Show {hiddenCount} more</span>
            <div className="inline-flex items-center justify-center w-5 h-5 bg-blue-200 dark:bg-blue-800 text-blue-800 dark:text-blue-200 rounded-full text-xs font-bold">
              {hiddenCount > 99 ? '99+' : hiddenCount}
            </div>
          </button>
        </div>
      )}

      {/* Collapse Controls - Show when expanded */}
      {shouldShowStacking && isExpanded && hiddenCount > 0 && (
        <div className="mb-3 flex justify-center">
          <button
            onClick={handleStackToggle}
            className={cn(
              "inline-flex items-center gap-2 px-3 py-2 rounded-full bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700",
              "text-gray-800 dark:text-gray-200 border border-gray-300 dark:border-gray-600",
              "transition-all duration-200 hover:scale-105 active:scale-95 text-sm font-medium shadow-sm hover:shadow-md",
              "min-h-[44px] lg:min-h-[32px] focus:outline-none focus:ring-2 focus:ring-gray-400 focus:ring-offset-2",
              "dark:focus:ring-gray-400 dark:focus:ring-offset-gray-900"
            )}
            aria-label="Collapse tool call stack"
            aria-expanded={true}
          >
            <ChevronDown className="w-4 h-4" />
            <span className="whitespace-nowrap">Show less</span>
          </button>
        </div>
      )}

      {/* Stacked Tool Calls */}
      <div className="relative" style={shouldShowStacking && !isExpanded ? { perspective: '1000px' } : {}}>
        {visibleTools.map((toolCall, index) => {
          const style = getStackStyle(index, visibleTools.length);
          
          return (
            <div
              key={toolCall.id}
              className={cn(
                "relative transition-all duration-300 ease-out"
              )}
              style={style}
            >
              <ConnectedToolChip
                toolCall={toolCall}
                onToggle={toggleExpanded}
                expanded={isChipExpanded(toolCall.id)}
                className={cn(
                  "transition-all duration-200",
                  // Add subtle shadow for depth effect when stacking
                  shouldShowStacking && !isExpanded && "shadow-lg hover:shadow-xl",
                  "relative"
                )}
              />
            </div>
          );
        })}
      </div>
    </div>
  );
});

StackedToolEvents.displayName = 'StackedToolEvents';

// Export individual chip components for backward compatibility
export { ConnectedToolChip, ConnectedToolChipGroup, useConnectedToolCalls } from './ConnectedToolChip';