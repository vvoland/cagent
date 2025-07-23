import React, { useState, useCallback, useRef, memo } from 'react';
import { cn } from '../../lib/utils';
import type { ToolChipProps, ChipState } from './types';
import { getToolIcon, getStatusIcon, getToolTypeFromName, getChipTheme } from './utils';
import { 
  ChevronRight, 
  Copy, 
  CheckCircle, 
  Maximize2,
  Minimize2
} from 'lucide-react';

const formatTimestamp = (date: Date): string => {
  return new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).format(date);
};

const formatArgs = (args: string): string => {
  try {
    const parsed = JSON.parse(args);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return args;
  }
};

const truncateText = (text: string, maxLength: number): string => {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
};

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
        "inline-flex items-center justify-center",
        "w-6 h-6 rounded transition-all duration-150",
        "hover:bg-black/10 dark:hover:bg-white/10",
        "focus:outline-none focus:ring-2 focus:ring-current focus:ring-offset-1",
        "active:scale-95",
        className
      )}
      aria-label={copied ? "Copied!" : "Copy to clipboard"}
      tabIndex={0}
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

export const ToolChip = memo<ToolChipProps>(({
  id,
  name,
  type: providedType,
  status,
  args,
  result,
  timestamp,
  variant = 'call',
  className,
  initialState = 'collapsed',
  onStateChange
}) => {
  const [state, setState] = useState<ChipState>(initialState);
  const [isAnimating, setIsAnimating] = useState(false);
  const [isFocused, setIsFocused] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);
  const chipRef = useRef<HTMLDivElement>(null);
  
  const toolType = providedType || getToolTypeFromName(name);
  const ToolIcon = getToolIcon(toolType);
  const StatusIcon = getStatusIcon(status);
  const theme = getChipTheme(toolType, variant);

  const handleStateChange = useCallback((newState: ChipState) => {
    if (newState === state) return;
    
    setIsAnimating(true);
    setState(newState);
    onStateChange?.(newState);
    
    // Reset animation state after transition
    setTimeout(() => setIsAnimating(false), 250);
  }, [state, onStateChange]);

  const handleClick = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    
    const newState: ChipState = 
      state === 'collapsed' ? 'preview' :
      state === 'preview' ? 'expanded' :
      'collapsed';
    
    handleStateChange(newState);
  }, [state, handleStateChange]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleClick(e as any);
    } else if (e.key === 'Escape' && state !== 'collapsed') {
      e.preventDefault();
      handleStateChange('collapsed');
    } else if (e.key === 'ArrowRight' && state === 'collapsed') {
      e.preventDefault();
      handleStateChange('preview');
    } else if (e.key === 'ArrowLeft' && state !== 'collapsed') {
      e.preventDefault();
      handleStateChange('collapsed');
    }
  }, [handleClick, state, handleStateChange]);

  const handleFocus = useCallback(() => {
    setIsFocused(true);
  }, []);

  const handleBlur = useCallback(() => {
    setIsFocused(false);
  }, []);

  const getContent = useCallback(() => {
    const content = variant === 'call' ? args : result;
    if (!content) return '';
    
    switch (state) {
      case 'preview':
        return truncateText(content, 100);
      case 'expanded':
        return formatArgs(content);
      default:
        return '';
    }
  }, [args, result, variant, state]);

  const getAnimationClass = useCallback(() => {
    if (!isAnimating) return '';
    
    switch (state) {
      case 'preview':
        return 'chip-preview';
      case 'expanded':
        return 'chip-expand';
      case 'collapsed':
        return 'chip-collapse';
      default:
        return '';
    }
  }, [state, isAnimating]);

  const displayName = variant === 'call' ? name : `${name}_result`;
  const truncatedId = id.length > 8 ? `${id.slice(0, 8)}...` : id;
  const hasContent = Boolean(args || result);

  return (
    <div
      ref={chipRef}
      className={cn(
        "tool-chip group relative inline-flex flex-col",
        "transition-all duration-200 ease-out transform-gpu",
        "rounded-lg border cursor-pointer select-none",
        "focus:outline-none min-h-[44px] lg:min-h-[28px]",
        "active:scale-[0.98] hover:scale-[1.02]",
        state === 'collapsed' && "h-7",
        state === 'preview' && "min-h-[2.5rem]",
        state === 'expanded' && "min-h-[4rem] max-h-[20rem]",
        theme.bg,
        theme.border,
        theme.hover,
        state === 'expanded' && "shadow-lg ring-1 ring-black/5 dark:ring-white/5",
        isFocused && "chip-focus-glow",
        getAnimationClass(),
        className
      )}
      onClick={handleClick}
      onKeyDown={handleKeyDown}
      onFocus={handleFocus}
      onBlur={handleBlur}
      tabIndex={0}
      role="button"
      aria-expanded={state !== 'collapsed'}
      aria-label={`${variant === 'call' ? 'Tool call' : 'Tool result'}: ${name}. Current state: ${state}. Press Enter to ${state === 'collapsed' ? 'expand to preview' : state === 'preview' ? 'expand to full view' : 'collapse'}. Use arrow keys to navigate states.`}
    >
      {/* Main chip content */}
      <div className="flex items-center justify-between px-2.5 py-1.5 min-h-[1.75rem]">
        <div className="flex items-center gap-2 min-w-0 flex-1">
          {/* Tool icon with status overlay */}
          <div className="relative flex-shrink-0">
            <ToolIcon className={cn("w-3.5 h-3.5", theme.icon)} />
            {/* Status indicator overlay */}
            <div className="absolute -top-0.5 -right-0.5">
              <StatusIcon 
                className={cn(
                  "w-2 h-2",
                  status === 'loading' && "animate-spin text-blue-500",
                  status === 'success' && "text-green-500",
                  status === 'error' && "text-red-500",
                  status === 'idle' && "text-gray-400 dark:text-gray-600"
                )} 
              />
            </div>
          </div>
          
          {/* Tool name and metadata */}
          <div className="flex flex-col min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <span className={cn(
                "text-xs font-semibold truncate",
                theme.text
              )}>
                {displayName}
              </span>
              {variant === 'result' && (
                <span className={cn(
                  "text-[10px] opacity-70 font-mono px-1 py-0.5 rounded",
                  "bg-black/10 dark:bg-white/10",
                  theme.text
                )}>
                  {truncatedId}
                </span>
              )}
            </div>
            
            {/* Preview content in collapsed state */}
            {state === 'collapsed' && hasContent && (
              <span className={cn(
                "text-[10px] opacity-60 truncate mt-0.5",
                theme.text
              )}>
                {truncateText(variant === 'call' ? args || '' : result || '', 40)}
              </span>
            )}
          </div>
          
          {/* Timestamp (only in expanded view) */}
          {timestamp && state === 'expanded' && (
            <span className={cn(
              "text-[10px] opacity-60 font-mono flex-shrink-0 px-1.5 py-0.5 rounded",
              "bg-black/5 dark:bg-white/5",
              theme.text
            )}>
              {formatTimestamp(timestamp)}
            </span>
          )}
        </div>

        {/* Right side controls */}
        <div className="flex items-center gap-1.5 flex-shrink-0">
          {/* Copy button (visible in preview/expanded) */}
          {(state === 'preview' || state === 'expanded') && hasContent && (
            <CopyButton 
              text={getContent()} 
              className={cn(
                "opacity-0 group-hover:opacity-100 transition-all duration-150",
                "scale-90 group-hover:scale-100",
                theme.text
              )}
            />
          )}

          {/* Expand/collapse indicator */}
          <div className="flex-shrink-0">
            {state === 'collapsed' ? (
              <ChevronRight className={cn(
                "w-3 h-3 transition-all duration-200",
                "group-hover:translate-x-0.5",
                theme.icon
              )} />
            ) : state === 'preview' ? (
              <Maximize2 className={cn(
                "w-3 h-3 transition-all duration-200",
                "group-hover:scale-110",
                theme.icon
              )} />
            ) : (
              <Minimize2 className={cn(
                "w-3 h-3 transition-all duration-200",
                "group-hover:scale-90",
                theme.icon
              )} />
            )}
          </div>
        </div>
      </div>

      {/* Expandable content */}
      {(state === 'preview' || state === 'expanded') && hasContent && (
        <div 
          ref={contentRef}
          className={cn(
            "px-2.5 pb-2 pt-0 transition-all duration-200",
            state === 'expanded' && "overflow-y-auto scrollbar-thin",
            "animate-slide-down"
          )}
        >
          <div className={cn(
            "text-[11px] font-mono p-2.5 rounded-md border",
            "bg-black/[0.03] dark:bg-white/[0.03]",
            "border-black/[0.08] dark:border-white/[0.08]",
            state === 'preview' && "line-clamp-3 overflow-hidden",
            state === 'expanded' && "whitespace-pre-wrap break-words max-h-48 overflow-y-auto"
          )}>
            {getContent()}
          </div>
          
          {/* Additional metadata in expanded view */}
          {state === 'expanded' && (
            <div className="mt-2.5 pt-2 border-t border-black/[0.08] dark:border-white/[0.08]">
              <div className="flex items-center justify-between text-[10px] opacity-70">
                <div className="flex items-center gap-2">
                  <span className={cn(
                    "px-1.5 py-0.5 rounded bg-black/5 dark:bg-white/5 font-medium",
                    theme.text
                  )}>
                    {toolType}
                  </span>
                  <span className={cn(
                    "px-1.5 py-0.5 rounded font-medium",
                    status === 'success' && "bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-300",
                    status === 'error' && "bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300",
                    status === 'loading' && "bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300",
                    status === 'idle' && "bg-gray-100 dark:bg-gray-900/30 text-gray-700 dark:text-gray-300"
                  )}>
                    {status}
                  </span>
                </div>
                {timestamp && (
                  <span className={cn("font-mono", theme.text)}>
                    {formatTimestamp(timestamp)}
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Loading overlay */}
      {status === 'loading' && (
        <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent dark:via-black/20 rounded-lg flex items-center justify-center backdrop-blur-[1px]">
          <div className="w-2 h-2 bg-current rounded-full animate-pulse" />
        </div>
      )}

      {/* Focus indicator */}
      {isFocused && (
        <div className="absolute inset-0 rounded-lg ring-2 ring-offset-2 ring-current pointer-events-none" />
      )}
    </div>
  );
});

ToolChip.displayName = 'ToolChip';