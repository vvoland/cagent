import { useCallback, useEffect, useRef } from 'react';

type KeyboardHandler = (event: KeyboardEvent) => void;

interface KeyboardShortcut {
  key: string;
  ctrlKey?: boolean;
  metaKey?: boolean;
  shiftKey?: boolean;
  altKey?: boolean;
  preventDefault?: boolean;
  handler: () => void;
  description?: string;
}

interface UseKeyboardOptions {
  enabled?: boolean;
  target?: HTMLElement | Document | null;
}

/**
 * Hook for handling keyboard shortcuts and events
 */
export const useKeyboard = (
  shortcuts: KeyboardShortcut[],
  options: UseKeyboardOptions = {}
) => {
  const { enabled = true, target = document } = options;
  const shortcutsRef = useRef(shortcuts);
  
  // Update shortcuts ref when shortcuts change
  shortcutsRef.current = shortcuts;

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;

    const { key, ctrlKey, metaKey, shiftKey, altKey } = event;
    
    shortcutsRef.current.forEach((shortcut) => {
      const keyMatches = shortcut.key.toLowerCase() === key.toLowerCase();
      const ctrlMatches = (shortcut.ctrlKey ?? false) === ctrlKey;
      const metaMatches = (shortcut.metaKey ?? false) === metaKey;
      const shiftMatches = (shortcut.shiftKey ?? false) === shiftKey;
      const altMatches = (shortcut.altKey ?? false) === altKey;

      if (keyMatches && ctrlMatches && metaMatches && shiftMatches && altMatches) {
        if (shortcut.preventDefault !== false) {
          event.preventDefault();
        }
        shortcut.handler();
      }
    });
  }, [enabled]);

  useEffect(() => {
    if (!target || !enabled) return;

    target.addEventListener('keydown', handleKeyDown as EventListener);
    
    return () => {
      target.removeEventListener('keydown', handleKeyDown as EventListener);
    };
  }, [target, enabled, handleKeyDown]);
};

/**
 * Hook for handling single key presses
 */
export const useKeyPress = (
  targetKey: string,
  handler: KeyboardHandler,
  options: UseKeyboardOptions = {}
) => {
  const { enabled = true, target = document } = options;
  const handlerRef = useRef(handler);
  
  // Update handler ref when handler changes
  handlerRef.current = handler;

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;
    
    if (event.key === targetKey) {
      handlerRef.current(event);
    }
  }, [targetKey, enabled]);

  useEffect(() => {
    if (!target || !enabled) return;

    target.addEventListener('keydown', handleKeyDown as EventListener);
    
    return () => {
      target.removeEventListener('keydown', handleKeyDown as EventListener);
    };
  }, [target, enabled, handleKeyDown]);
};

/**
 * Hook for handling key combinations
 */
export const useKeyCombination = (
  keys: string[],
  handler: () => void,
  options: UseKeyboardOptions = {}
) => {
  const { enabled = true, target = document } = options;
  const pressedKeys = useRef(new Set<string>());
  const handlerRef = useRef(handler);
  
  // Update handler ref when handler changes
  handlerRef.current = handler;

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;
    
    pressedKeys.current.add(event.key.toLowerCase());
    
    // Check if all required keys are pressed
    const allKeysPressed = keys.every(key => 
      pressedKeys.current.has(key.toLowerCase())
    );
    
    if (allKeysPressed && pressedKeys.current.size === keys.length) {
      event.preventDefault();
      handlerRef.current();
    }
  }, [keys, enabled]);

  const handleKeyUp = useCallback((event: KeyboardEvent) => {
    if (!enabled) return;
    
    pressedKeys.current.delete(event.key.toLowerCase());
  }, [enabled]);

  useEffect(() => {
    if (!target || !enabled) return;

    target.addEventListener('keydown', handleKeyDown as EventListener);
    target.addEventListener('keyup', handleKeyUp as EventListener);
    
    return () => {
      target.removeEventListener('keydown', handleKeyDown as EventListener);
      target.removeEventListener('keyup', handleKeyUp as EventListener);
      pressedKeys.current.clear();
    };
  }, [target, enabled, handleKeyDown, handleKeyUp]);
};

/**
 * Common keyboard shortcuts for web applications
 */
export const commonShortcuts = {
  // Navigation
  newSession: { key: 'n', ctrlKey: true, description: 'Create new session' },
  search: { key: 'k', ctrlKey: true, description: 'Search' },
  
  // Editing
  save: { key: 's', ctrlKey: true, description: 'Save' },
  undo: { key: 'z', ctrlKey: true, description: 'Undo' },
  redo: { key: 'y', ctrlKey: true, description: 'Redo' },
  
  // UI
  toggleTheme: { key: 'd', ctrlKey: true, description: 'Toggle dark mode' },
  toggleSidebar: { key: 'b', ctrlKey: true, description: 'Toggle sidebar' },
  
  // Accessibility
  focusSearch: { key: '/', description: 'Focus search' },
  escape: { key: 'Escape', description: 'Close modal or cancel' },
  
  // Copy/Paste
  copy: { key: 'c', ctrlKey: true, description: 'Copy' },
  paste: { key: 'v', ctrlKey: true, description: 'Paste' },
};