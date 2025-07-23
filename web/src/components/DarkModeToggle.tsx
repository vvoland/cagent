import { Moon, Sun } from "lucide-react";
import { Button } from "./ui/button";
import { useDarkMode } from "../hooks/useDarkMode";
import { memo } from "react";

export const DarkModeToggle = memo(() => {
  const { isDarkMode, toggleDarkMode } = useDarkMode();

  return (
    <Button
      variant="outline"
      size="icon"
      onClick={toggleDarkMode}
      className="h-9 w-9 transition-all hover:scale-105 active:scale-95"
      title={isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
      aria-label={isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
    >
      <div className="relative">
        {isDarkMode ? (
          <Sun className="h-4 w-4 transition-all" />
        ) : (
          <Moon className="h-4 w-4 transition-all" />
        )}
        <span className="sr-only">
          {isDarkMode ? "Switch to light mode" : "Switch to dark mode"}
        </span>
      </div>
    </Button>
  );
});

DarkModeToggle.displayName = 'DarkModeToggle';