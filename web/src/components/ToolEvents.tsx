import { memo } from "react";
import { 
  ConnectedToolChip, 
  ConnectedToolChipGroup, 
  useConnectedToolCalls 
} from "./ConnectedToolChip";

// Legacy individual event components - kept for backward compatibility
interface ToolCallEventProps {
  name: string;
  args: string;
}

interface ToolResultEventProps {
  id: string;
  content: string;
}

// Legacy components that create simple connected tool calls
export const ToolCallEvent = memo<ToolCallEventProps>(({ name, args }) => {
  const mockEvents = [{ type: 'tool_call' as const, name, args }];
  const { connectedToolCalls, toggleExpanded, isExpanded } = useConnectedToolCalls(mockEvents);
  
  if (connectedToolCalls.length === 0) return null;
  
  const toolCall = connectedToolCalls[0];
  if (!toolCall) return null;
  
  return (
    <ConnectedToolChipGroup className="mx-2 lg:mx-3">
      <ConnectedToolChip
        toolCall={toolCall}
        onToggle={toggleExpanded}
        expanded={isExpanded(toolCall.id)}
        className="transition-all hover:shadow-md hover:scale-[1.005] active:scale-100"
      />
    </ConnectedToolChipGroup>
  );
});

ToolCallEvent.displayName = 'ToolCallEvent';

export const ToolResultEvent = memo<ToolResultEventProps>(() => {
  // This component is now handled by the connected system
  // It should not be rendered separately as results are connected to calls
  return null;
});

ToolResultEvent.displayName = 'ToolResultEvent';

// New main component for rendering connected tool operations
interface ConnectedToolEventsProps {
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
}

export const ConnectedToolEvents = memo<ConnectedToolEventsProps>(({ events, className }) => {
  const { connectedToolCalls, toggleExpanded, isExpanded } = useConnectedToolCalls(events);
  
  if (connectedToolCalls.length === 0) return null;
  
  return (
    <ConnectedToolChipGroup className={className || ''}>
      {connectedToolCalls.map((toolCall) => (
        <ConnectedToolChip
          key={toolCall.id}
          toolCall={toolCall}
          onToggle={toggleExpanded}
          expanded={isExpanded(toolCall.id)}
          className="transition-all hover:shadow-md hover:scale-[1.005] active:scale-100"
        />
      ))}
    </ConnectedToolChipGroup>
  );
});

ConnectedToolEvents.displayName = 'ConnectedToolEvents';

// Export both the connected system and stacked system for easy usage
export { ConnectedToolChip, ConnectedToolChipGroup, useConnectedToolCalls } from './ConnectedToolChip';
export { StackedToolEvents } from './StackedToolEvents';