// Import ToolType from utils now since it's defined there
import type { ToolType } from './utils';

export type ChipState = 'collapsed' | 'preview' | 'expanded';

export type ChipStatus = 'idle' | 'loading' | 'success' | 'error';

export interface ToolChipProps {
  id: string;
  name: string;
  type: ToolType | null;
  status: ChipStatus;
  args?: string;
  result?: string;
  timestamp?: Date;
  variant?: 'call' | 'result';
  className?: string;
  initialState?: ChipState;
  onStateChange?: (state: ChipState) => void;
}

export interface ToolChipGroupProps {
  children: React.ReactNode;
  className?: string;
  maxVisibleChips?: number;
  orientation?: 'horizontal' | 'vertical';
}

// Re-export ToolType for convenience
export type { ToolType } from './utils';