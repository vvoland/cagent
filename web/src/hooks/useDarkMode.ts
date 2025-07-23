import { useState, useEffect, useCallback, useMemo } from "react";

interface UseDarkModeReturn {
  isDarkMode: boolean;
  toggleDarkMode: () => void;
  setDarkMode: (isDark: boolean) => void;
}

const THEME_KEY = "theme";
const DARK_THEME = "dark";
const LIGHT_THEME = "light";
const DARK_CLASS = "dark";

export const useDarkMode = (): UseDarkModeReturn => {
  // Memoize the initial theme detection to avoid repeated calculations
  const getInitialTheme = useMemo(() => {
    try {
      // Check localStorage first
      const savedTheme = localStorage.getItem(THEME_KEY);
      if (savedTheme === DARK_THEME || savedTheme === LIGHT_THEME) {
        return savedTheme === DARK_THEME;
      }
    } catch (error) {
      // localStorage might not be available (SSR, private mode, etc.)
      console.warn("Failed to access localStorage:", error);
    }

    // Fall back to system preference
    try {
      return window.matchMedia("(prefers-color-scheme: dark)").matches;
    } catch (error) {
      // matchMedia might not be available
      console.warn("Failed to access matchMedia:", error);
      return false; // Default to light mode
    }
  }, []);

  const [isDarkMode, setIsDarkModeState] = useState(getInitialTheme);

  // Memoize the theme application logic
  const applyTheme = useCallback((isDark: boolean) => {
    try {
      const root = document.documentElement;
      const themeValue = isDark ? DARK_THEME : LIGHT_THEME;

      if (isDark) {
        root.classList.add(DARK_CLASS);
      } else {
        root.classList.remove(DARK_CLASS);
      }

      localStorage.setItem(THEME_KEY, themeValue);
    } catch (error) {
      console.warn("Failed to apply theme:", error);
    }
  }, []);

  // Memoize the toggle function
  const toggleDarkMode = useCallback(() => {
    setIsDarkModeState(prev => {
      const newValue = !prev;
      applyTheme(newValue);
      return newValue;
    });
  }, [applyTheme]);

  // Memoize the direct setter function
  const setDarkMode = useCallback((isDark: boolean) => {
    if (isDark !== isDarkMode) {
      setIsDarkModeState(isDark);
      applyTheme(isDark);
    }
  }, [isDarkMode, applyTheme]);

  // Apply theme on mount and when isDarkMode changes
  useEffect(() => {
    applyTheme(isDarkMode);
  }, [isDarkMode, applyTheme]);

  // Listen for system theme changes
  useEffect(() => {
    let mediaQuery: MediaQueryList | null = null;
    let cleanup: (() => void) | null = null;

    try {
      mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
      
      const handleSystemThemeChange = (e: MediaQueryListEvent) => {
        // Only update if user hasn't explicitly set a theme
        try {
          const savedTheme = localStorage.getItem(THEME_KEY);
          if (!savedTheme) {
            setDarkMode(e.matches);
          }
        } catch (error) {
          // If we can't access localStorage, still respond to system changes
          setDarkMode(e.matches);
        }
      };

      // Use the modern addEventListener if available, fallback to deprecated addListener
      if (mediaQuery.addEventListener) {
        mediaQuery.addEventListener("change", handleSystemThemeChange);
        cleanup = () => mediaQuery!.removeEventListener("change", handleSystemThemeChange);
      } else if (mediaQuery.addListener) {
        // Deprecated but still supported in some browsers
        mediaQuery.addListener(handleSystemThemeChange);
        cleanup = () => mediaQuery!.removeListener(handleSystemThemeChange);
      }
    } catch (error) {
      console.warn("Failed to set up system theme listener:", error);
    }

    // Cleanup function
    return () => {
      if (cleanup) {
        try {
          cleanup();
        } catch (error) {
          console.warn("Failed to cleanup theme listener:", error);
        }
      }
    };
  }, [setDarkMode]);

  // Memoize the return object to prevent unnecessary re-renders
  const returnValue = useMemo((): UseDarkModeReturn => ({
    isDarkMode,
    toggleDarkMode,
    setDarkMode,
  }), [isDarkMode, toggleDarkMode, setDarkMode]);

  return returnValue;
};