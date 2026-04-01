# Contextual Help Dialog

## Overview

The cagent TUI now includes a contextual help dialog that displays all currently active key bindings. This makes it easy to discover available keyboard shortcuts without leaving the application.

## Usage

Press **F1** to open the help dialog. (Alternatively, **Ctrl+?** also works in terminals that support keyboard enhancements.)

The dialog will show all key bindings that are currently active based on:
- Which panel is focused (content view vs editor)
- Whether lean mode is enabled
- Terminal capabilities (keyboard enhancements support)
- Current UI state (history search active, etc.)

## Features

### Contextual Bindings
The help dialog is **context-aware** - it shows only the key bindings that are valid for your current state:

- **When editor is focused**: Shows editor-specific bindings (Ctrl+g for external editor, Ctrl+r for history search, etc.)
- **When content is focused**: Shows content view bindings (navigation, inline editing, etc.)
- **Always visible**: Global bindings like Ctrl+c (quit), Tab (switch focus), Ctrl+k (commands)

### Automatic Categorization
Key bindings are automatically grouped into three categories for easy scanning:

1. **General** - Common keys like Esc, Enter, Tab
2. **Control Key Shortcuts** - All Ctrl+ combinations
3. **Other** - Any remaining bindings

### Scrollable
If there are more key bindings than fit on screen, the dialog is scrollable:
- Use **↑/↓** arrow keys or **PgUp/PgDn** to scroll
- Press **Esc** or **Enter** to close the dialog

## Implementation Details

The help dialog leverages the existing `Bindings()` method structure in the TUI, which already returns different key bindings based on context. This means:

- The help is always accurate and up-to-date
- No manual maintenance of help text needed
- Automatically adapts to new features and key bindings

### For Developers

To add new key bindings to the help:

1. Create your `key.Binding` with `.WithHelp()` to provide description
2. Add it to the appropriate `Bindings()` method (app, component, or page level)
3. It will automatically appear in the help dialog

Example:
```go
newFeatureKey := key.NewBinding(
    key.WithKeys("ctrl+f"),
    key.WithHelp("Ctrl+f", "new feature"),
)
```

The help dialog will pick this up automatically when it's included in any `Bindings()` return value.

## Key Binding Notes

**F1** is used as the primary help key because it's universally supported across all terminals. **Ctrl+?** (Ctrl+Shift+/) is also supported as an alternative, but requires terminals with keyboard enhancement support (kitty protocol).

## Related Files

- `pkg/tui/dialog/help.go` - Help dialog implementation
- `pkg/tui/tui.go` - Key binding handler (Ctrl+? binding)
- `pkg/tui/dialog/readonly_scroll_dialog.go` - Base dialog with scrolling support
