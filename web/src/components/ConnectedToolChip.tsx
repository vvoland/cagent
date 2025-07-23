import React, { useState, useCallback, memo } from 'react';
import { cn } from '../lib/utils';
import { 
  FileText, 
  Search, 
  Terminal, 
  Globe, 
  Database, 
  Zap, 
  BarChart3, 
  Brain,
  Settings,
  CheckCircle,
  XCircle,
  Clock,
  ChevronRight,
  ChevronDown,
  Copy,
  AlertCircle
} from 'lucide-react';

// Types
type ToolType = 'file' | 'search' | 'shell' | 'web' | 'database' | 'api' | 'analysis' | 'memory' | 'default';
type ToolCallStatus = 'pending' | 'success' | 'error' | 'timeout';

interface ToolCallData {
  name: string;
  args: string;
  timestamp?: Date;
}

interface ToolResultData {
  id: string;
  content: string;
  success?: boolean;
  timestamp?: Date;
}

interface ConnectedToolCall {
  callEvent: ToolCallData;
  resultEvent?: ToolResultData;
  status: ToolCallStatus;
  id: string;
}

interface ConnectedToolChipProps {
  toolCall: ConnectedToolCall;
  className?: string;
  onToggle?: (id: string, expanded: boolean) => void;
  expanded?: boolean;
}

// Utility functions
const getToolTypeFromName = (toolName: string): ToolType => {
  const name = toolName.toLowerCase();
  
  if (name.includes('file') || name.includes('read') || name.includes('write') || name.includes('edit')) {
    return 'file';
  }
  if (name.includes('search') || name.includes('find') || name.includes('grep')) {
    return 'search';
  }
  if (name.includes('shell') || name.includes('bash') || name.includes('cmd') || name.includes('exec')) {
    return 'shell';
  }
  if (name.includes('web') || name.includes('http') || name.includes('fetch') || name.includes('url')) {
    return 'web';
  }
  if (name.includes('database') || name.includes('db') || name.includes('sql') || name.includes('query')) {
    return 'database';
  }
  if (name.includes('api') || name.includes('request') || name.includes('post') || name.includes('get')) {
    return 'api';
  }
  if (name.includes('analyze') || name.includes('analysis') || name.includes('chart') || name.includes('report')) {
    return 'analysis';
  }
  if (name.includes('memory') || name.includes('remember') || name.includes('store') || name.includes('recall')) {
    return 'memory';
  }
  
  return 'default';
};

const getToolIcon = (toolType: ToolType) => {
  switch (toolType) {
    case 'file': return FileText;
    case 'search': return Search;
    case 'shell': return Terminal;
    case 'web': return Globe;
    case 'database': return Database;
    case 'api': return Zap;
    case 'analysis': return BarChart3;
    case 'memory': return Brain;
    default: return Settings;
  }
};

const getStatusIcon = (status: ToolCallStatus) => {
  switch (status) {
    case 'success': return CheckCircle;
    case 'error': return XCircle;
    case 'timeout': return AlertCircle;
    default: return Clock;
  }
};

const getGroupTheme = (toolType: ToolType, status: ToolCallStatus) => {
  const baseColors = {
    file: 'border-blue-200 dark:border-blue-800',
    search: 'border-purple-200 dark:border-purple-800',
    shell: 'border-gray-200 dark:border-gray-800',
    web: 'border-indigo-200 dark:border-indigo-800',
    database: 'border-teal-200 dark:border-teal-800',
    api: 'border-yellow-200 dark:border-yellow-800',
    analysis: 'border-green-200 dark:border-green-800',
    memory: 'border-pink-200 dark:border-pink-800',
    default: 'border-slate-200 dark:border-slate-800'
  };

  const statusOverlay = {
    success: 'bg-green-50/50 dark:bg-green-950/30',
    error: 'bg-red-50/50 dark:bg-red-950/30',
    timeout: 'bg-orange-50/50 dark:bg-orange-950/30',
    pending: 'bg-blue-50/50 dark:bg-blue-950/30'
  };

  return cn(
    'bg-white dark:bg-gray-900',
    baseColors[toolType] || baseColors.default,
    statusOverlay[status]
  );
};

const getCallChipTheme = (toolType: ToolType) => {
  const themes = {
    file: 'bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200 border-blue-300 dark:border-blue-700',
    search: 'bg-purple-100 dark:bg-purple-900/50 text-purple-800 dark:text-purple-200 border-purple-300 dark:border-purple-700',
    shell: 'bg-gray-100 dark:bg-gray-900/50 text-gray-800 dark:text-gray-200 border-gray-300 dark:border-gray-700',
    web: 'bg-indigo-100 dark:bg-indigo-900/50 text-indigo-800 dark:text-indigo-200 border-indigo-300 dark:border-indigo-700',
    database: 'bg-teal-100 dark:bg-teal-900/50 text-teal-800 dark:text-teal-200 border-teal-300 dark:border-teal-700',
    api: 'bg-yellow-100 dark:bg-yellow-900/50 text-yellow-800 dark:text-yellow-200 border-yellow-300 dark:border-yellow-700',
    analysis: 'bg-green-100 dark:bg-green-900/50 text-green-800 dark:text-green-200 border-green-300 dark:border-green-700',
    memory: 'bg-pink-100 dark:bg-pink-900/50 text-pink-800 dark:text-pink-200 border-pink-300 dark:border-pink-700',
    default: 'bg-slate-100 dark:bg-slate-900/50 text-slate-800 dark:text-slate-200 border-slate-300 dark:border-slate-700'
  };
  
  return themes[toolType] || themes.default;
};

const getResultChipTheme = (status: ToolCallStatus) => {
  const themes = {
    success: 'bg-green-100 dark:bg-green-900/50 text-green-800 dark:text-green-200 border-green-300 dark:border-green-700',
    error: 'bg-red-100 dark:bg-red-900/50 text-red-800 dark:text-red-200 border-red-300 dark:border-red-700',
    timeout: 'bg-orange-100 dark:bg-orange-900/50 text-orange-800 dark:text-orange-200 border-orange-300 dark:border-orange-700',
    pending: 'bg-blue-100 dark:bg-blue-900/50 text-blue-800 dark:text-blue-200 border-blue-300 dark:border-blue-700'
  };
  
  return themes[status];
};

const formatJSON = (jsonString: string): string => {
  try {
    const parsed = JSON.parse(jsonString);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return jsonString;
  }
};

const truncateText = (text: string, maxLength: number): string => {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
};

// Copy button component
const CopyButton = memo<{ text: string; className?: string }>(({ text, className }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation();
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy text:', error);
    }
  }, [text]);

  return (
    <button
      onClick={handleCopy}
      className={cn(
        "inline-flex items-center justify-center w-6 h-6 rounded transition-all duration-150",
        "hover:bg-black/10 dark:hover:bg-white/10 focus:outline-none focus:ring-2 focus:ring-current focus:ring-offset-1",
        "active:scale-95", className
      )}
      aria-label={copied ? "Copied!" : "Copy to clipboard"}
    >
      {copied ? (
        <CheckCircle className="w-3 h-3 text-green-600 dark:text-green-400" />
      ) : (
        <Copy className="w-3 h-3" />
      )}
    </button>
  );
});

CopyButton.displayName = 'CopyButton';

// Main component
export const ConnectedToolChip = memo<ConnectedToolChipProps>(({
  toolCall,
  className,
  onToggle,
  expanded = false
}) => {
  const toolType = getToolTypeFromName(toolCall.callEvent.name);
  const ToolIcon = getToolIcon(toolType);
  const StatusIcon = getStatusIcon(toolCall.status);
  
  const handleToggle = useCallback(() => {
    onToggle?.(toolCall.id, !expanded);
  }, [onToggle, toolCall.id, expanded]);

  const getStatusText = () => {
    switch (toolCall.status) {
      case 'success':
        return toolCall.resultEvent ? 'Success' : 'Completed';
      case 'error':
        return 'Error';
      case 'timeout':
        return 'Timeout';
      default:
        return 'Processing...';
    }
  };

  const getStatusMessage = () => {
    switch (toolCall.status) {
      case 'success':
        return toolCall.resultEvent?.content || 'Tool execution completed successfully.';
      case 'error':
        return 'Tool execution failed.';
      case 'timeout':
        return 'Tool execution timed out.';
      default:
        return 'Tool is currently processing...';
    }
  };

  const getResultPreview = () => {
    if (!toolCall.resultEvent) return '';
    const content = toolCall.resultEvent.content;
    return truncateText(content, 40);
  };

  return (
    <div
      className={cn(
        "rounded-lg border transition-all duration-200 overflow-hidden",
        "hover:shadow-sm hover:scale-[1.005]",
        getGroupTheme(toolType, toolCall.status),
        className
      )}
    >
      {/* Header - Always visible */}
      <div
        className="flex items-center justify-between p-3 cursor-pointer select-none"
        onClick={handleToggle}
        role="button"
        tabIndex={0}
        aria-expanded={expanded}
        aria-label={`Tool operation: ${toolCall.callEvent.name}. Status: ${getStatusText()}. Click to ${expanded ? 'collapse' : 'expand'}`}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleToggle();
          }
        }}
      >
        <div className="flex items-center gap-3 min-w-0 flex-1">
          <ToolIcon className="w-4 h-4 flex-shrink-0 text-current" />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold truncate">
                {toolCall.callEvent.name}
              </span>
              <StatusIcon className={cn(
                "w-3.5 h-3.5 flex-shrink-0",
                toolCall.status === 'success' && "text-green-600 dark:text-green-400",
                toolCall.status === 'error' && "text-red-600 dark:text-red-400",
                toolCall.status === 'timeout' && "text-orange-600 dark:text-orange-400",
                toolCall.status === 'pending' && "text-blue-600 dark:text-blue-400 animate-pulse"
              )} />
            </div>
            {!expanded && (
              <div className="text-xs text-current/70 truncate mt-0.5">
                {getStatusText()}{toolCall.resultEvent && `: ${getResultPreview()}`}
              </div>
            )}
          </div>
        </div>
        
        <div className="flex items-center gap-1 flex-shrink-0">
          {expanded ? (
            <ChevronDown className="w-4 h-4 transition-transform duration-200" />
          ) : (
            <ChevronRight className="w-4 h-4 transition-transform duration-200" />
          )}
        </div>
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div className="px-3 pb-3 space-y-3 border-t border-current/10">
          {/* Tool Call Section */}
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-semibold text-current/80 uppercase tracking-wide">
                Tool Call
              </h4>
              <CopyButton text={toolCall.callEvent.args} />
            </div>
            <div className={cn(
              "p-2.5 rounded-md border text-xs",
              getCallChipTheme(toolType)
            )}>
              <div className="font-mono whitespace-pre-wrap break-words">
                {formatJSON(toolCall.callEvent.args)}
              </div>
            </div>
          </div>

          {/* Result Section */}
          {toolCall.resultEvent && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <h4 className="text-xs font-semibold text-current/80 uppercase tracking-wide flex items-center gap-1.5">
                  Result
                  <StatusIcon className={cn(
                    "w-3 h-3",
                    toolCall.status === 'success' && "text-green-600 dark:text-green-400",
                    toolCall.status === 'error' && "text-red-600 dark:text-red-400",
                    toolCall.status === 'timeout' && "text-orange-600 dark:text-orange-400"
                  )} />
                </h4>
                <CopyButton text={toolCall.resultEvent.content} />
              </div>
              <div className={cn(
                "p-2.5 rounded-md border text-xs max-h-48 overflow-y-auto",
                getResultChipTheme(toolCall.status)
              )}>
                <div className="font-mono whitespace-pre-wrap break-words">
                  {toolCall.resultEvent.content}
                </div>
              </div>
            </div>
          )}

          {/* Processing/No Result State */}
          {!toolCall.resultEvent && (
            <div className="space-y-2">
              <h4 className="text-xs font-semibold text-current/80 uppercase tracking-wide">
                {toolCall.status === 'pending' ? 'Processing' : 'Result'}
              </h4>
              <div className={cn(
                "p-2.5 rounded-md border text-xs flex items-center gap-2",
                getResultChipTheme(toolCall.status)
              )}>
                {toolCall.status === 'pending' ? (
                  <>
                    <Clock className="w-3 h-3 animate-pulse" />
                    <span className="text-current/70">Processing tool execution...</span>
                  </>
                ) : (
                  <>
                    <StatusIcon className={cn(
                      "w-3 h-3",
                      toolCall.status === 'success' && "text-green-600 dark:text-green-400"
                    )} />
                    <span className="text-current/90">{getStatusMessage()}</span>
                  </>
                )}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
});

ConnectedToolChip.displayName = 'ConnectedToolChip';

// Group container for multiple connected tool chips
export const ConnectedToolChipGroup = memo<{ 
  children: React.ReactNode; 
  className?: string;
}>(({ children, className }) => {
  return (
    <div className={cn("space-y-2", className)}>
      {children}
    </div>
  );
});

ConnectedToolChipGroup.displayName = 'ConnectedToolChipGroup';

// Hook for managing connected tool calls
export const useConnectedToolCalls = (events: Array<{ type: 'tool_call' | 'tool_result'; [key: string]: any }>) => {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  const connectedToolCalls = React.useMemo(() => {
    const calls: ConnectedToolCall[] = [];
    let currentId = 0;

    for (let i = 0; i < events.length; i++) {
      const event = events[i];
      
      if (event && event.type === 'tool_call') {
        const id = `tool-${currentId++}`;
        const toolCall: ConnectedToolCall = {
          id,
          callEvent: {
            name: event.name || 'unknown',
            args: event.args || '{}',
            timestamp: event.timestamp
          },
          status: 'pending'
        };

        // Look ahead for the result
        const nextEvent = events[i + 1];
        if (nextEvent && nextEvent.type === 'tool_result') {
          toolCall.resultEvent = {
            id: nextEvent.id || 'unknown',
            content: nextEvent.content || '',
            success: nextEvent.success,
            timestamp: nextEvent.timestamp
          };
          toolCall.status = nextEvent.success !== false ? 'success' : 'error';
          i++; // Skip the result event in the next iteration
        }

        calls.push(toolCall);
      }
    }

    return calls;
  }, [events]);

  const toggleExpanded = useCallback((id: string, expanded: boolean) => {
    setExpandedIds(prev => {
      const newSet = new Set(prev);
      if (expanded) {
        newSet.add(id);
      } else {
        newSet.delete(id);
      }
      return newSet;
    });
  }, []);

  return {
    connectedToolCalls,
    expandedIds,
    toggleExpanded,
    isExpanded: (id: string) => expandedIds.has(id)
  };
};