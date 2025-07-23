import React, { memo, useMemo } from 'react';
import { cn } from '../../lib/utils';
import type { ToolChipGroupProps } from './types';

export const ToolChipGroup = memo<ToolChipGroupProps>(({
  children,
  className,
  maxVisibleChips = 10,
  orientation = 'horizontal'
}) => {
  const childrenArray = React.Children.toArray(children);
  const visibleChildren = childrenArray.slice(0, maxVisibleChips);
  const hiddenCount = childrenArray.length - maxVisibleChips;

  const containerClasses = useMemo(() => cn(
    "flex gap-1.5 items-start",
    orientation === 'horizontal' ? "flex-wrap" : "flex-col",
    "transition-all duration-200",
    className
  ), [orientation, className]);

  return (
    <div className={containerClasses} role="group" aria-label="Tool execution chips">
      {visibleChildren}
      
      {hiddenCount > 0 && (
        <div className={cn(
          "inline-flex items-center px-2 py-1 rounded-md border",
          "bg-muted/50 border-muted text-muted-foreground",
          "text-xs font-medium cursor-help",
          "min-h-[1.75rem] transition-colors",
          "hover:bg-muted hover:border-muted-foreground/30"
        )}>
          +{hiddenCount} more
        </div>
      )}
    </div>
  );
});

ToolChipGroup.displayName = 'ToolChipGroup';