# Message Action Buttons - Design Specifications

## Overview
Design specifications for message action buttons (copy and replay) in the chat interface. The buttons integrate with the existing MessageEvent component while maintaining Nord theme consistency and accessibility standards.

## Requirements Summary
- **Copy button**: Available for ALL messages (user and agent) - copies message content to clipboard
- **Replay button**: Available for USER messages ONLY - resends the same message  
- **Discoverability**: Subtle but discoverable (hover states + always visible on mobile)
- **Touch targets**: 44px minimum on mobile, 32px on desktop
- **Theming**: Full Nord theme integration (dark/light mode)
- **Accessibility**: WCAG 2.1 AA compliant with proper ARIA labels
- **Placement**: Three options analyzed - top-right corner (recommended)

## Design Concepts

### Option 1: Top-Right Corner Placement (RECOMMENDED)
```
┌─────────────────────────────────────────────────────┐
│ 🤖 agent-name    [user]                    [⋯] [↻]  │
│                                                      │
│ This is the message content with markdown            │
│ formatting that can span multiple lines...           │
│                                                      │
└─────────────────────────────────────────────────────┘
```

**Advantages:**
- Clean, familiar pattern (similar to Discord, Slack)
- Doesn't interfere with content readability
- Easy hover discovery
- Consistent spacing and alignment

### Option 2: Floating Action Buttons
```
┌─────────────────────────────────────────────────────┐
│ 🤖 agent-name    [user]                              │
│                                           ┌───┐     │
│ This is the message content...            │ ⋯ │     │
│                                           │ ↻ │     │
│                                           └───┘     │
└─────────────────────────────────────────────────────┘
```

**Disadvantages:**
- Can overlap with content on narrow screens
- Floating elements can be disruptive
- Harder to make accessible

### Option 3: Inline with Header
```
┌─────────────────────────────────────────────────────┐
│ 🤖 agent-name    [user]    [Copy] [Replay]          │
│                                                      │
│ This is the message content with markdown            │
│ formatting that can span multiple lines...           │
│                                                      │
└─────────────────────────────────────────────────────┘
```

**Disadvantages:**
- Takes up header space
- Can cause wrapping on mobile
- Always visible (not subtle)

## Recommended Design: Top-Right Corner Placement

### Desktop Wireframe
```
Desktop Layout (>= 1024px):
┌────────────────────────────────────────────────────────────────────┐
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ 🤖 Assistant    [assistant]              [Copy] [Replay]    │   │ 
│  │                                              32px   32px     │   │
│  │ Here's a detailed response with **markdown** formatting.    │   │
│  │                                                             │   │
│  │ ```javascript                                               │   │
│  │ const example = "code block";                               │   │
│  │ ```                                                         │   │
│  │                                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ 👤 User    [user]                         [Copy]           │   │
│  │                                             32px            │   │
│  │ What is the weather like today?                             │   │
│  │                                                             │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘

Hover States:
┌─────────────────────────────────────────────────────────────────┐
│ 🤖 Assistant    [assistant]              [Copy] [Replay]       │
│                                          ^^^^^^^ ^^^^^^^        │
│                                          Visible on hover      │
└─────────────────────────────────────────────────────────────────┘
```

### Mobile Wireframe  
```
Mobile Layout (< 768px):
┌────────────────────────────────────┐
│                                    │
│ ┌────────────────────────────────┐ │
│ │ 🤖 Assistant  [assistant] [⋯][↻]│ │
│ │                        44px 44px│ │
│ │                                 │ │
│ │ Here's a response with          │ │
│ │ **markdown** formatting.        │ │
│ │                                 │ │
│ │ ```js                           │ │
│ │ const example = "code";         │ │
│ │ ```                             │ │
│ │                                 │ │
│ └────────────────────────────────┘ │
│                                    │
│ ┌────────────────────────────────┐ │
│ │ 👤 User  [user]           [⋯]  │ │
│ │                         44px   │ │
│ │                                │ │
│ │ What is the weather like?      │ │
│ │                                │ │
│ └────────────────────────────────┘ │
│                                    │
└────────────────────────────────────┘

Always Visible:
- Buttons always shown on mobile
- Larger touch targets (44px)
- Icons instead of text labels
```

## Visual Design Specifications

### Button Styles

#### Desktop Buttons (32px)
```css
.message-action-button {
  /* Base Styles */
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid transparent;
  transition: all 0.2s ease-in-out;
  cursor: pointer;
  
  /* Typography */
  font-size: 14px;
  font-weight: 500;
  
  /* Icon sizing */
  svg {
    width: 16px;
    height: 16px;
  }
}

/* Light Mode Colors */
.message-action-button {
  background: rgba(229, 231, 235, 0.1); /* gray-200/10 */
  color: #6b7280; /* gray-500 */
  border-color: transparent;
}

.message-action-button:hover {
  background: rgba(229, 231, 235, 0.8); /* gray-200/80 */
  color: #374151; /* gray-700 */
  border-color: rgba(156, 163, 175, 0.3); /* gray-400/30 */
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.message-action-button:active {
  transform: translateY(0);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

/* Dark Mode Colors (Nord Theme) */
.dark .message-action-button {
  background: rgba(76, 86, 106, 0.1); /* nord1/10 */
  color: #d8dee9; /* nord4 */
  border-color: transparent;
}

.dark .message-action-button:hover {
  background: rgba(76, 86, 106, 0.3); /* nord1/30 */
  color: #eceff4; /* nord6 */
  border-color: rgba(129, 161, 193, 0.2); /* nord10/20 */
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
}
```

#### Mobile Buttons (44px)
```css
@media (max-width: 768px) {
  .message-action-button {
    width: 44px;
    height: 44px;
    border-radius: 8px;
    
    /* Larger icons for mobile */
    svg {
      width: 20px;
      height: 20px;
    }
  }
  
  /* Always visible on mobile */
  .message-actions {
    opacity: 1;
  }
}
```

### Button Variants

#### Copy Button
```css
.copy-button {
  /* Success feedback state */
  &.copied {
    background: rgba(34, 197, 94, 0.1); /* green-500/10 */
    color: #22c55e; /* green-500 */
    border-color: rgba(34, 197, 94, 0.2);
  }
  
  .dark &.copied {
    background: rgba(163, 190, 140, 0.2); /* nord14/20 */
    color: #a3be8c; /* nord14 */
    border-color: rgba(163, 190, 140, 0.3);
  }
}
```

#### Replay Button  
```css
.replay-button {
  /* Only show for user messages */
  &.user-only {
    display: flex;
  }
  
  &.agent-message {
    display: none;
  }
  
  /* Loading state */
  &.loading {
    pointer-events: none;
    opacity: 0.6;
    
    svg {
      animation: spin 1s linear infinite;
    }
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
```

### Layout Integration

#### Header Layout Updates
```css
.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 8px;
  
  .header-info {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0; /* Allow shrinking */
  }
  
  .message-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.2s ease-in-out;
  }
  
  /* Show on hover for desktop */
  @media (min-width: 769px) {
    &:hover .message-actions {
      opacity: 1;
    }
  }
  
  /* Always show on mobile */
  @media (max-width: 768px) {
    .message-actions {
      opacity: 1;
    }
  }
}
```

## Icon Specifications

### Copy Button Icons
- **Default**: `Copy` icon (clipboard outline)
- **Success**: `Check` icon (checkmark) - shown for 2 seconds after successful copy
- **Error**: `AlertCircle` icon - shown if copy fails

### Replay Button Icons  
- **Default**: `RotateCcw` icon (counterclockwise arrow)
- **Loading**: `Loader2` icon with spin animation
- **Only visible**: For user messages (role === "user")

### Icon Library
Using Lucide React icons for consistency:
```javascript
import { Copy, Check, AlertCircle, RotateCcw, Loader2 } from 'lucide-react';
```

## Accessibility Specifications

### ARIA Labels
```html
<!-- Copy Button -->
<button
  aria-label="Copy message content to clipboard"
  aria-describedby="copy-feedback-{messageId}"
  role="button"
  tabindex="0"
>
  <Copy aria-hidden="true" />
  <span className="sr-only">Copy</span>
</button>

<!-- Copy Success Feedback -->
<div 
  id="copy-feedback-{messageId}" 
  aria-live="polite" 
  className="sr-only"
>
  {copied ? "Message copied to clipboard" : ""}
</div>

<!-- Replay Button -->
<button
  aria-label="Resend this message"
  aria-describedby="replay-feedback-{messageId}"
  role="button"
  tabindex="0"
  disabled={isLoading}
>
  <RotateCcw aria-hidden="true" />
  <span className="sr-only">Replay</span>
</button>
```

### Keyboard Navigation
- **Tab Order**: Copy button → Replay button (if visible)
- **Enter/Space**: Activates button
- **Focus Indicators**: Clear 2px outline with theme-appropriate colors
```css
.message-action-button:focus-visible {
  outline: 2px solid #3b82f6; /* blue-500 */
  outline-offset: 2px;
}

.dark .message-action-button:focus-visible {
  outline-color: #81a1c1; /* nord10 */
}
```

### Screen Reader Support
- Buttons announce their purpose clearly
- Success/error states communicated via aria-live regions
- Icons marked as `aria-hidden="true"`
- Text alternatives provided via `sr-only` classes

## Responsive Behavior

### Breakpoint Strategy
```css
/* Mobile First Approach */
.message-actions {
  /* Mobile: Always visible, larger targets */
  opacity: 1;
  gap: 8px;
}

.message-action-button {
  width: 44px;
  height: 44px;
  border-radius: 8px;
}

/* Tablet */
@media (min-width: 640px) {
  .message-actions {
    gap: 6px;
  }
  
  .message-action-button {
    width: 36px;
    height: 36px;
    border-radius: 6px;
  }
}

/* Desktop */
@media (min-width: 1024px) {
  .message-actions {
    opacity: 0; /* Hide by default */
    gap: 4px;
  }
  
  .message-header:hover .message-actions {
    opacity: 1; /* Show on hover */
  }
  
  .message-action-button {
    width: 32px;
    height: 32px;
  }
}
```

### Content Overflow Handling
- Header uses `justify-content: space-between` to push actions to the right
- Info section uses `flex: 1` and `min-width: 0` to allow shrinking
- Actions use `flex-shrink: 0` to maintain size
- Long agent names get truncated with ellipsis

## Animation Specifications

### Hover Animations
```css
.message-action-button {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.message-action-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.message-action-button:active {
  transform: translateY(0);
  transition: transform 0.1s;
}
```

### Success Animations
```css
@keyframes success-pulse {
  0% { transform: scale(1); }
  50% { transform: scale(1.1); }
  100% { transform: scale(1); }
}

.copy-button.copied {
  animation: success-pulse 0.3s ease-out;
}
```

### Fade In/Out
```css
.message-actions {
  transition: opacity 0.2s ease-in-out;
}

/* Reduced motion support */
@media (prefers-reduced-motion: reduce) {
  .message-action-button {
    transition: none;
  }
  
  .message-action-button:hover {
    transform: none;
  }
  
  .copy-button.copied {
    animation: none;
  }
}
```

## Implementation Notes

### State Management
- Use React state for copy feedback (2-second timeout)
- Use React state for replay loading state
- Consider memoization for button click handlers

### Performance Considerations
- Debounce rapid button clicks
- Use `React.memo` for button components
- Lazy load icons if bundle size is a concern

### Error Handling
- Graceful fallback if clipboard API not available
- Toast notifications for copy/replay feedback
- Retry mechanism for failed replay attempts

### Testing Requirements
- Keyboard navigation testing
- Screen reader testing with NVDA/JAWS
- Touch target testing on actual mobile devices
- Cross-browser clipboard API testing

This design provides a clean, accessible, and mobile-friendly solution that integrates seamlessly with your existing Nord-themed chat interface.