import { memo, useCallback, useEffect, useRef } from "react";
import { Button } from "./ui/button";
import { X } from "lucide-react";
import { cn } from "../lib/utils";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  className?: string;
}

export const Modal = memo<ModalProps>(({ isOpen, onClose, title, children, className }) => {
  const modalRef = useRef<HTMLDivElement>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);

  // Handle escape key press
  const handleEscapeKey = useCallback((event: KeyboardEvent) => {
    if (event.key === 'Escape') {
      onClose();
    }
  }, [onClose]);

  // Handle focus management
  useEffect(() => {
    if (isOpen) {
      // Store the currently focused element
      previousFocusRef.current = document.activeElement as HTMLElement;
      
      // Add escape key listener
      document.addEventListener('keydown', handleEscapeKey);
      
      // Focus the modal
      setTimeout(() => {
        modalRef.current?.focus();
      }, 0);
      
      // Prevent body scroll
      document.body.style.overflow = 'hidden';
    } else {
      // Remove escape key listener
      document.removeEventListener('keydown', handleEscapeKey);
      
      // Restore body scroll
      document.body.style.overflow = '';
      
      // Restore focus to the previous element
      if (previousFocusRef.current) {
        previousFocusRef.current.focus();
      }
    }

    // Cleanup
    return () => {
      document.removeEventListener('keydown', handleEscapeKey);
      document.body.style.overflow = '';
    };
  }, [isOpen, handleEscapeKey]);

  // Handle backdrop click
  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <div 
      className="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 dark:bg-background/80 max-h-full animate-in fade-in-0 duration-200 p-4 lg:p-0"
      onClick={handleBackdropClick}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div 
        ref={modalRef}
        className={cn(
          "fixed left-[50%] top-[50%] z-50 grid w-full max-w-lg translate-x-[-50%] translate-y-[-50%] gap-4",
          "border bg-background p-4 lg:p-6 shadow-lg duration-200 rounded-lg",
          "dark:border-border dark:bg-background dark:shadow-xl",
          "max-h-[90vh] overflow-y-auto",
          "animate-in slide-in-from-left-1/2 slide-in-from-top-[48%] duration-200",
          "mx-4 lg:mx-0", // Add horizontal margins on mobile
          className
        )}
        tabIndex={-1}
        role="document"
      >
        <div className="flex items-center justify-between sticky top-0 bg-background pb-2 border-b border-border/50">
          <h3 
            id="modal-title"
            className="text-base lg:text-lg font-semibold dark:text-foreground line-clamp-1 pr-2"
          >
            {title}
          </h3>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 lg:h-6 lg:w-6 rounded-md flex-shrink-0 hover:bg-destructive/10 hover:text-destructive transition-colors"
            onClick={onClose}
            aria-label="Close modal"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
        <div className="relative flex-1 min-h-0 text-sm lg:text-base">
          {children}
        </div>
      </div>
    </div>
  );
});

Modal.displayName = 'Modal';