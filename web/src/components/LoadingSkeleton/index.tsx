import { memo } from 'react';
import { cn } from '../../lib/utils';

interface SkeletonProps {
  className?: string;
  width?: string | number;
  height?: string | number;
  variant?: 'rectangular' | 'circular' | 'rounded';
  animation?: 'pulse' | 'wave' | 'none';
}

export const Skeleton = memo<SkeletonProps>(({ 
  className, 
  width, 
  height, 
  variant = 'rounded',
  animation = 'pulse'
}) => {
  const baseClasses = 'bg-muted';
  
  const variantClasses = {
    rectangular: '',
    circular: 'rounded-full',
    rounded: 'rounded-md',
  };

  const animationClasses = {
    pulse: 'animate-pulse',
    wave: 'animate-shimmer',
    none: '',
  };

  const style: React.CSSProperties = {};
  if (width) style.width = typeof width === 'number' ? `${width}px` : width;
  if (height) style.height = typeof height === 'number' ? `${height}px` : height;

  return (
    <div
      className={cn(
        baseClasses,
        variantClasses[variant],
        animationClasses[animation],
        className
      )}
      style={style}
      aria-hidden="true"
    />
  );
});

Skeleton.displayName = 'Skeleton';

// Specific skeleton components for common use cases
export const MessageSkeleton = memo(() => (
  <div className="rounded-lg p-4 shadow-md dark:shadow-lg space-y-3">
    <div className="flex items-center gap-2 mb-2">
      <Skeleton width={20} height={20} variant="circular" />
      <Skeleton width={120} height={16} />
      <Skeleton width={60} height={12} />
    </div>
    <div className="space-y-2">
      <Skeleton width="100%" height={14} />
      <Skeleton width="85%" height={14} />
      <Skeleton width="92%" height={14} />
      <Skeleton width="70%" height={14} />
    </div>
  </div>
));

MessageSkeleton.displayName = 'MessageSkeleton';

export const SessionSkeleton = memo(() => (
  <div className="p-3 rounded-lg space-y-2">
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Skeleton width={16} height={16} variant="circular" />
        <Skeleton width={100} height={16} />
      </div>
      <Skeleton width={16} height={16} variant="circular" />
    </div>
    <div className="flex items-center justify-between">
      <Skeleton width={60} height={12} />
      <Skeleton width={40} height={12} />
    </div>
  </div>
));

SessionSkeleton.displayName = 'SessionSkeleton';

export const ToolEventSkeleton = memo(() => (
  <div className="inline-flex items-center gap-2 m-3">
    <Skeleton width={16} height={16} variant="circular" />
    <Skeleton width={80} height={16} />
  </div>
));

ToolEventSkeleton.displayName = 'ToolEventSkeleton';

// Skeleton list component
interface SkeletonListProps {
  count?: number;
  component: React.ComponentType;
  className?: string;
}

export const SkeletonList = memo<SkeletonListProps>(({ 
  count = 3, 
  component: Component, 
  className 
}) => (
  <div className={cn('space-y-4', className)}>
    {Array.from({ length: count }, (_, index) => (
      <Component key={`skeleton-${index}`} />
    ))}
  </div>
));

SkeletonList.displayName = 'SkeletonList';