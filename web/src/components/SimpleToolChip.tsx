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
  ChevronRight,
  Copy,
  ChevronDown
} from 'lucide-react';

// Simple, self-contained types
type ToolType = 'file' | 'search' | 'shell' | 'web' | 'database' | 'api' | 'analysis' | 'memory' | 'default';
type ChipState = 'collapsed' | 'expanded';

interface SimpleToolChipProps {
  name: string;
  args?: string;
  result?: string;
  variant: 'call' | 'result';
  className?: string;
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

const getChipTheme = (toolType: ToolType, variant: 'call' | 'result') => {
  const baseThemes = {
    file: variant === 'call' ? 'bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800 text-blue-800 dark:text-blue-200' 
                              : 'bg-blue-100 dark:bg-blue-900/50 border-blue-300 dark:border-blue-700 text-blue-900 dark:text-blue-100',
    search: variant === 'call' ? 'bg-purple-50 dark:bg-purple-950/30 border-purple-200 dark:border-purple-800 text-purple-800 dark:text-purple-200'
                                : 'bg-purple-100 dark:bg-purple-900/50 border-purple-300 dark:border-purple-700 text-purple-900 dark:text-purple-100',
    shell: variant === 'call' ? 'bg-gray-50 dark:bg-gray-950/30 border-gray-200 dark:border-gray-800 text-gray-800 dark:text-gray-200'
                               : 'bg-gray-100 dark:bg-gray-900/50 border-gray-300 dark:border-gray-700 text-gray-900 dark:text-gray-100',
    web: variant === 'call' ? 'bg-indigo-50 dark:bg-indigo-950/30 border-indigo-200 dark:border-indigo-800 text-indigo-800 dark:text-indigo-200'
                             : 'bg-indigo-100 dark:bg-indigo-900/50 border-indigo-300 dark:border-indigo-700 text-indigo-900 dark:text-indigo-100',
    database: variant === 'call' ? 'bg-teal-50 dark:bg-teal-950/30 border-teal-200 dark:border-teal-800 text-teal-800 dark:text-teal-200'
                                  : 'bg-teal-100 dark:bg-teal-900/50 border-teal-300 dark:border-teal-700 text-teal-900 dark:text-teal-100',
    api: variant === 'call' ? 'bg-yellow-50 dark:bg-yellow-950/30 border-yellow-200 dark:border-yellow-800 text-yellow-800 dark:text-yellow-200'
                             : 'bg-yellow-100 dark:bg-yellow-900/50 border-yellow-300 dark:border-yellow-700 text-yellow-900 dark:text-yellow-100',
    analysis: variant === 'call' ? 'bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800 text-green-800 dark:text-green-200'
                                  : 'bg-green-100 dark:bg-green-900/50 border-green-300 dark:border-green-700 text-green-900 dark:text-green-100',
    memory: variant === 'call' ? 'bg-pink-50 dark:bg-pink-950/30 border-pink-200 dark:border-pink-800 text-pink-800 dark:text-pink-200'
                                : 'bg-pink-100 dark:bg-pink-900/50 border-pink-300 dark:border-pink-700 text-pink-900 dark:text-pink-100',
    default: variant === 'call' ? 'bg-slate-50 dark:bg-slate-950/30 border-slate-200 dark:border-slate-800 text-slate-800 dark:text-slate-200'
                                 : 'bg-slate-100 dark:bg-slate-900/50 border-slate-300 dark:border-slate-700 text-slate-900 dark:text-slate-100'
  };
  
  return baseThemes[toolType] || baseThemes.default;
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
export const SimpleToolChip = memo<SimpleToolChipProps>(({
  name,
  args,
  result,
  variant,
  className
}) => {
  const [state, setState] = useState<ChipState>('collapsed');
  
  const toolType = getToolTypeFromName(name);
  const ToolIcon = getToolIcon(toolType);
  const theme = getChipTheme(toolType, variant);
  const content = variant === 'call' ? args : result;
  const displayName = variant === 'call' ? name : `${name} result`;

  const handleToggle = useCallback(() => {
    setState(current => current === 'collapsed' ? 'expanded' : 'collapsed');
  }, []);

  const getDisplayContent = useCallback(() => {
    if (!content) return '';
    if (state === 'collapsed') {
      return truncateText(content, 40);
    }
    return formatJSON(content);
  }, [content, state]);

  return (
    <div className="flex gap-1.5 items-start">
      <div
        className={cn(
          "inline-flex flex-col rounded-lg border cursor-pointer select-none transition-all duration-200",
          "min-h-[28px] lg:min-h-[24px]",
          state === 'collapsed' ? "max-w-xs" : "min-w-[300px] max-w-2xl",
          theme,
          "hover:shadow-sm hover:scale-[1.01]",
          className
        )}
        onClick={handleToggle}
        role="button"
        tabIndex={0}
        aria-expanded={state === 'expanded'}
        aria-label={`${variant === 'call' ? 'Tool call' : 'Tool result'}: ${name}. Click to ${state === 'collapsed' ? 'expand' : 'collapse'}`}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handleToggle();
          }
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-2.5 py-1.5 min-h-[1.75rem]">
          <div className="flex items-center gap-2 min-w-0 flex-1">
            <ToolIcon className="w-3.5 h-3.5 flex-shrink-0" />
            <span className="text-xs font-semibold truncate">
              {displayName}
            </span>
            {state === 'collapsed' && content && (
              <span className="text-[10px] opacity-60 truncate">
                {truncateText(content, 20)}
              </span>
            )}
          </div>
          
          <div className="flex items-center gap-1 flex-shrink-0">
            {state === 'expanded' && content && (
              <CopyButton 
                text={getDisplayContent()} 
                className="opacity-0 group-hover:opacity-100"
              />
            )}
            {state === 'collapsed' ? (
              <ChevronRight className="w-3 h-3 transition-transform duration-200" />
            ) : (
              <ChevronDown className="w-3 h-3 transition-transform duration-200" />
            )}
          </div>
        </div>

        {/* Expanded content */}
        {state === 'expanded' && content && (
          <div className="px-2.5 pb-2 pt-0">
            <div className="text-[11px] font-mono p-2.5 rounded-md border bg-black/[0.03] dark:bg-white/[0.03] border-black/[0.08] dark:border-white/[0.08] whitespace-pre-wrap break-words max-h-48 overflow-y-auto">
              {getDisplayContent()}
            </div>
          </div>
        )}
      </div>
    </div>
  );
});

SimpleToolChip.displayName = 'SimpleToolChip';

// Group component
export const SimpleToolChipGroup = memo<{ children: React.ReactNode; className?: string }>(({
  children,
  className
}) => {
  return (
    <div className={cn("flex flex-wrap gap-1.5 items-start", className)}>
      {children}
    </div>
  );
});

SimpleToolChipGroup.displayName = 'SimpleToolChipGroup';