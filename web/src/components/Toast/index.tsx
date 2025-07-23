import { createContext, useContext, useCallback, useState, type ReactNode, memo } from 'react';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Button } from '../ui/button';

export type ToastType = 'success' | 'error' | 'info' | 'warning';

export interface Toast {
  id: string;
  type: ToastType;
  title: string;
  description?: string;
  duration?: number;
  action?: {
    label: string;
    onClick: () => void;
  };
}

interface ToastContextType {
  toasts: Toast[];
  addToast: (toast: Omit<Toast, 'id'>) => string;
  removeToast: (id: string) => void;
  removeAllToasts: () => void;
}

const ToastContext = createContext<ToastContextType | undefined>(undefined);

export const useToast = () => {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error('useToast must be used within a ToastProvider');
  }
  return context;
};

// Toast icons
const toastIcons = {
  success: CheckCircle,
  error: AlertCircle,
  info: Info,
  warning: AlertTriangle,
};

// Toast styles
const toastStyles = {
  success: 'border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950/30 text-green-800 dark:text-green-200',
  error: 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/30 text-red-800 dark:text-red-200',
  info: 'border-blue-200 dark:border-blue-800 bg-blue-50 dark:bg-blue-950/30 text-blue-800 dark:text-blue-200',
  warning: 'border-yellow-200 dark:border-yellow-800 bg-yellow-50 dark:bg-yellow-950/30 text-yellow-800 dark:text-yellow-200',
};

// Individual toast component
const ToastComponent = memo<{ toast: Toast; onRemove: (id: string) => void }>(({ toast, onRemove }) => {
  const Icon = toastIcons[toast.type];
  
  const handleRemove = useCallback(() => {
    onRemove(toast.id);
  }, [toast.id, onRemove]);

  const handleAction = useCallback(() => {
    if (toast.action) {
      toast.action.onClick();
    }
  }, [toast.action]);

  return (
    <div
      className={cn(
        'pointer-events-auto relative flex w-full max-w-sm items-start gap-3 rounded-lg border p-4 shadow-lg transition-all',
        'animate-slide-up',
        toastStyles[toast.type]
      )}
      role="alert"
      aria-live="polite"
    >
      <Icon className="h-5 w-5 flex-shrink-0 mt-0.5" />
      
      <div className="flex-1 min-w-0">
        <div className="font-semibold text-sm">{toast.title}</div>
        {toast.description && (
          <div className="text-sm opacity-90 mt-1">{toast.description}</div>
        )}
        {toast.action && (
          <Button
            variant="ghost"
            size="sm"
            onClick={handleAction}
            className="mt-2 h-auto p-0 text-current hover:bg-current/10"
          >
            {toast.action.label}
          </Button>
        )}
      </div>
      
      <Button
        variant="ghost"
        size="icon"
        onClick={handleRemove}
        className="h-6 w-6 text-current hover:bg-current/10 flex-shrink-0"
        aria-label="Close notification"
      >
        <X className="h-4 w-4" />
      </Button>
    </div>
  );
});

ToastComponent.displayName = 'ToastComponent';

// Toast container
const ToastContainer = memo<{ toasts: Toast[]; onRemove: (id: string) => void }>(({ toasts, onRemove }) => {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none">
      {toasts.map((toast) => (
        <ToastComponent key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  );
});

ToastContainer.displayName = 'ToastContainer';

// Toast provider
export const ToastProvider = memo<{ children: ReactNode }>(({ children }) => {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const addToast = useCallback((toast: Omit<Toast, 'id'>) => {
    const id = Math.random().toString(36).substr(2, 9);
    const newToast: Toast = {
      ...toast,
      id,
      duration: toast.duration ?? 5000,
    };

    setToasts((prev) => [...prev, newToast]);

    // Auto remove toast after duration
    if (newToast.duration && newToast.duration > 0) {
      setTimeout(() => {
        removeToast(id);
      }, newToast.duration);
    }

    return id;
  }, []);

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const removeAllToasts = useCallback(() => {
    setToasts([]);
  }, []);

  return (
    <ToastContext.Provider value={{ toasts, addToast, removeToast, removeAllToasts }}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  );
});

ToastProvider.displayName = 'ToastProvider';

// Convenience hooks for different toast types
export const useToastHelpers = () => {
  const { addToast } = useToast();

  return {
    success: useCallback((title: string, description?: string, options?: Partial<Toast>) => {
      const toastData: Omit<Toast, 'id'> = { ...options, type: 'success', title };
      if (description !== undefined) {
        toastData.description = description;
      }
      return addToast(toastData);
    }, [addToast]),
    
    error: useCallback((title: string, description?: string, options?: Partial<Toast>) => {
      const toastData: Omit<Toast, 'id'> = { ...options, type: 'error', title };
      if (description !== undefined) {
        toastData.description = description;
      }
      return addToast(toastData);
    }, [addToast]),
    
    info: useCallback((title: string, description?: string, options?: Partial<Toast>) => {
      const toastData: Omit<Toast, 'id'> = { ...options, type: 'info', title };
      if (description !== undefined) {
        toastData.description = description;
      }
      return addToast(toastData);
    }, [addToast]),
    
    warning: useCallback((title: string, description?: string, options?: Partial<Toast>) => {
      const toastData: Omit<Toast, 'id'> = { ...options, type: 'warning', title };
      if (description !== undefined) {
        toastData.description = description;
      }
      return addToast(toastData);
    }, [addToast]),
  };
};