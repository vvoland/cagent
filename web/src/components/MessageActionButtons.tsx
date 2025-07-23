import { useState, useCallback, memo } from 'react';
import { Copy, Check, AlertCircle, RotateCcw, Loader2 } from 'lucide-react';
import { cn } from '../lib/utils';

interface MessageActionButtonsProps {
  content: string;
  role: string;
  onReplay?: (() => void) | undefined;
  className?: string;
}

interface CopyButtonProps {
  content: string;
  className?: string;
}

interface ReplayButtonProps {
  onReplay: () => void;
  isLoading?: boolean;
  className?: string;
}

// Copy Button Component
const CopyButton = memo<CopyButtonProps>(({ content, className }) => {
  const [copyState, setCopyState] = useState<'idle' | 'success' | 'error'>('idle');

  const handleCopy = useCallback(async () => {
    try {
      // Modern clipboard API with fallback
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(content);
      } else {
        // Fallback for older browsers or non-secure contexts
        const textArea = document.createElement('textarea');
        textArea.value = content;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        textArea.style.top = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        document.execCommand('copy');
        textArea.remove();
      }
      
      setCopyState('success');
      
      // Reset state after 2 seconds
      setTimeout(() => {
        setCopyState('idle');
      }, 2000);
      
    } catch (error) {
      console.error('Failed to copy message:', error);
      setCopyState('error');
      
      // Reset error state after 2 seconds
      setTimeout(() => {
        setCopyState('idle');
      }, 2000);
    }
  }, [content]);

  const getIcon = () => {
    switch (copyState) {
      case 'success':
        return <Check className="w-4 h-4 lg:w-[16px] lg:h-[16px]" aria-hidden="true" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 lg:w-[16px] lg:h-[16px]" aria-hidden="true" />;
      default:
        return <Copy className="w-4 h-4 lg:w-[16px] lg:h-[16px]" aria-hidden="true" />;
    }
  };

  const getAriaLabel = () => {
    switch (copyState) {
      case 'success':
        return 'Message copied to clipboard';
      case 'error':
        return 'Failed to copy message';
      default:
        return 'Copy message content to clipboard';
    }
  };

  return (
    <>
      <button
        onClick={handleCopy}
        className={cn(
          'message-action-button copy-button',
          // Base styles
          'w-11 h-11 lg:w-8 lg:h-8 rounded-lg lg:rounded-md',
          'flex items-center justify-center',
          'border border-transparent transition-all duration-200 ease-in-out',
          'cursor-pointer focus:outline-none',
          
          // Light mode styles
          'bg-gray-200/10 text-gray-500 hover:bg-gray-200/80 hover:text-gray-700',
          'hover:border-gray-400/30 hover:shadow-sm hover:-translate-y-px',
          'active:translate-y-0 active:shadow-sm',
          
          // Dark mode styles
          'dark:bg-[#4c566a]/10 dark:text-[#d8dee9]',
          'dark:hover:bg-[#4c566a]/30 dark:hover:text-[#eceff4]',
          'dark:hover:border-[#81a1c1]/20 dark:hover:shadow-lg dark:hover:shadow-black/30',
          
          // Focus styles
          'focus-visible:outline-2 focus-visible:outline-blue-500 focus-visible:outline-offset-2',
          'dark:focus-visible:outline-[#81a1c1]',
          
          // State-specific styles
          copyState === 'success' && [
            'bg-green-500/10 text-green-500 border-green-500/20',
            'dark:bg-[#a3be8c]/20 dark:text-[#a3be8c] dark:border-[#a3be8c]/30',
            'animate-pulse'
          ],
          copyState === 'error' && [
            'bg-red-500/10 text-red-500 border-red-500/20',
            'dark:bg-[#bf616a]/20 dark:text-[#bf616a] dark:border-[#bf616a]/30'
          ],
          
          className
        )}
        aria-label={getAriaLabel()}
        aria-describedby={`copy-feedback-${Date.now()}`}
        disabled={copyState !== 'idle'}
      >
        {getIcon()}
        <span className="sr-only">Copy</span>
      </button>
      
      {/* Screen reader feedback */}
      <div 
        id={`copy-feedback-${Date.now()}`}
        aria-live="polite" 
        className="sr-only"
      >
        {copyState === 'success' && 'Message copied to clipboard'}
        {copyState === 'error' && 'Failed to copy message'}
      </div>
    </>
  );
});

CopyButton.displayName = 'CopyButton';

// Replay Button Component
const ReplayButton = memo<ReplayButtonProps>(({ onReplay, isLoading = false, className }) => {
  const handleReplay = useCallback(() => {
    if (!isLoading && onReplay) {
      onReplay();
    }
  }, [onReplay, isLoading]);

  return (
    <button
      onClick={handleReplay}
      disabled={isLoading}
      className={cn(
        'message-action-button replay-button',
        // Base styles
        'w-11 h-11 lg:w-8 lg:h-8 rounded-lg lg:rounded-md',
        'flex items-center justify-center',
        'border border-transparent transition-all duration-200 ease-in-out',
        'cursor-pointer focus:outline-none',
        
        // Light mode styles
        'bg-gray-200/10 text-gray-500 hover:bg-gray-200/80 hover:text-gray-700',
        'hover:border-gray-400/30 hover:shadow-sm hover:-translate-y-px',
        'active:translate-y-0 active:shadow-sm',
        
        // Dark mode styles
        'dark:bg-[#4c566a]/10 dark:text-[#d8dee9]',
        'dark:hover:bg-[#4c566a]/30 dark:hover:text-[#eceff4]',
        'dark:hover:border-[#81a1c1]/20 dark:hover:shadow-lg dark:hover:shadow-black/30',
        
        // Focus styles
        'focus-visible:outline-2 focus-visible:outline-blue-500 focus-visible:outline-offset-2',
        'dark:focus-visible:outline-[#81a1c1]',
        
        // Disabled/loading styles
        isLoading && [
          'pointer-events-none opacity-60',
          'cursor-not-allowed'
        ],
        
        // Reduced motion support
        'motion-reduce:transition-none motion-reduce:transform-none',
        
        className
      )}
      aria-label={isLoading ? "Replaying message..." : "Resend this message"}
      aria-describedby={`replay-feedback-${Date.now()}`}
    >
      {isLoading ? (
        <Loader2 className="w-4 h-4 lg:w-[16px] lg:h-[16px] animate-spin" aria-hidden="true" />
      ) : (
        <RotateCcw className="w-4 h-4 lg:w-[16px] lg:h-[16px]" aria-hidden="true" />
      )}
      <span className="sr-only">{isLoading ? 'Replaying' : 'Replay'}</span>
    </button>
  );
});

ReplayButton.displayName = 'ReplayButton';

// Main MessageActionButtons Component
export const MessageActionButtons = memo<MessageActionButtonsProps>(({ 
  content, 
  role, 
  onReplay, 
  className 
}) => {
  const [isReplaying, setIsReplaying] = useState(false);
  
  const handleReplay = useCallback(async () => {
    if (!onReplay) return;
    
    setIsReplaying(true);
    try {
      await onReplay();
    } catch (error) {
      console.error('Failed to replay message:', error);
    } finally {
      setIsReplaying(false);
    }
  }, [onReplay]);

  const isUserMessage = role.toLowerCase() === 'user';

  return (
    <div 
      className={cn(
        'message-actions',
        'flex gap-1 lg:gap-1 flex-shrink-0',
        // Always visible on mobile, hover-visible on desktop
        'opacity-100 lg:opacity-0 lg:group-hover:opacity-100',
        'transition-opacity duration-200 ease-in-out',
        className
      )}
    >
      {/* Copy button - always visible for all messages */}
      <CopyButton content={content} />
      
      {/* Replay button - only visible for user messages */}
      {isUserMessage && onReplay && (
        <ReplayButton 
          onReplay={handleReplay} 
          isLoading={isReplaying}
        />
      )}
    </div>
  );
});

MessageActionButtons.displayName = 'MessageActionButtons';

export default MessageActionButtons;