import React, { useState, useRef, useEffect, memo, useCallback } from 'react';
import { ChevronDown, ChevronUp } from 'lucide-react';
import { cn } from '../lib/utils';

interface CollapsibleContentProps {
  children: React.ReactNode;
  maxLines?: number;
  isLatest?: boolean;
  className?: string;
}

export const CollapsibleContent = memo<CollapsibleContentProps>(({
  children,
  maxLines = 5,
  isLatest = false,
  className
}) => {
  const [isExpanded, setIsExpanded] = useState(isLatest);
  const [shouldCollapse, setShouldCollapse] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);

  // Simplified content overflow detection using line-clamp
  useEffect(() => {
    // Always expand latest messages
    if (isLatest) {
      setShouldCollapse(false);
      setIsExpanded(true);
      return;
    }

    const checkIfContentOverflows = () => {
      const element = contentRef.current;
      if (!element) return;

      // Create a temporary clone to measure height without affecting the original
      const clone = element.cloneNode(true) as HTMLElement;
      clone.style.position = 'absolute';
      clone.style.visibility = 'hidden';
      clone.style.top = '-9999px';
      clone.style.left = '-9999px';
      clone.style.width = element.offsetWidth + 'px';
      clone.style.height = 'auto';
      clone.style.maxHeight = 'none';
      clone.style.display = 'block';
      clone.style.webkitLineClamp = 'unset';
      clone.style.webkitBoxOrient = 'unset';
      clone.style.overflow = 'visible';
      
      document.body.appendChild(clone);
      const fullHeight = clone.offsetHeight;
      
      // Now apply line clamp to measure clamped height
      clone.style.display = '-webkit-box';
      clone.style.webkitLineClamp = maxLines.toString();
      clone.style.webkitBoxOrient = 'vertical';
      clone.style.overflow = 'hidden';
      
      const clampedHeight = clone.offsetHeight;
      
      // Clean up
      document.body.removeChild(clone);
      
      // If full height is greater than clamped height, content overflows
      const isOverflowing = fullHeight > clampedHeight + 10; // Add 10px buffer
      
      setShouldCollapse(isOverflowing);
      
      // If content overflows and this isn't the latest message, collapse by default
      if (isOverflowing && !isLatest) {
        setIsExpanded(false);
      } else {
        setIsExpanded(true);
      }
    };

    // Use a short delay to ensure ReactMarkdown content is fully rendered
    const timeoutId = setTimeout(checkIfContentOverflows, 300);
    
    return () => clearTimeout(timeoutId);
  }, [children, maxLines, isLatest]);

  const toggleExpanded = useCallback(() => {
    setIsExpanded(prev => !prev);
  }, []);

  // If it's the latest message or shouldn't collapse, render normally
  if (isLatest || !shouldCollapse) {
    return (
      <div 
        ref={contentRef}
        className={className}
      >
        {children}
      </div>
    );
  }

  return (
    <div className={cn("relative overflow-hidden", className)}>
      {/* Content container */}
      <div className="relative">
        {/* Content with optional line-clamp */}
        <div
          ref={contentRef}
          className={cn(
            "transition-all duration-300 ease-in-out",
            !isExpanded && [
              "line-clamp-5", // Use Tailwind's line-clamp utility
              "overflow-hidden",
              "relative",
            ].join(" ")
          )}
          style={{
            // Fallback for browsers that don't support line-clamp-5
            ...((!isExpanded && maxLines === 5) ? {} : {
              display: isExpanded ? 'block' : '-webkit-box',
              WebkitLineClamp: isExpanded ? 'unset' : maxLines,
              WebkitBoxOrient: isExpanded ? 'unset' : 'vertical',
              overflow: isExpanded ? 'visible' : 'hidden',
            })
          }}
        >
          {children}
        </div>
        
      </div>
      
      {/* Button container - separate from content to avoid affecting layout */}
      <div className="flex justify-end mt-3">
        <button
          onClick={toggleExpanded}
          className={cn(
            "inline-flex mb-1 mr-1 items-right gap-2 px-3 py-1.5 rounded-md text-sm font-medium",
            "bg-secondary hover:bg-secondary/80 dark:bg-secondary dark:hover:bg-secondary/80",
            "text-muted-foreground hover:text-foreground",
            "border border-border hover:border-primary/30",
            "transition-all duration-200 hover:shadow-sm hover:scale-[1.02]",
            "focus:outline-none focus:ring-2 focus:ring-primary/50 focus:ring-offset-1",
            "min-h-[32px] lg:min-h-[28px]", // Touch-friendly on mobile
            "shadow-sm" // Add slight shadow so button stands out
          )}
          aria-label={isExpanded ? "Show less content" : "Show more content"}
          aria-expanded={isExpanded}
        >
          {isExpanded ? (
            <>
              <span>Show less</span>
              <ChevronUp className="w-4 h-4" />
            </>
          ) : (
            <>
              <span>Show more</span>
              <ChevronDown className="w-4 h-4" />
            </>
          )}
        </button>
      </div>
    </div>
  );
});

CollapsibleContent.displayName = 'CollapsibleContent';