import React from 'react';
import { StackedToolEvents } from './StackedToolEvents';
import { MessageEvent } from './MessageEvents';
import { CollapsibleContent } from './CollapsibleContent';

// Demo component to test the new stacked UI functionality
export const TestStackedUI: React.FC = () => {
  // Mock tool events for testing stacking
  const mockToolEvents = [
    {
      type: 'tool_call' as const,
      name: 'search_files',
      args: JSON.stringify({ 
        pattern: '*.tsx', 
        query: 'React components',
        directory: '/src/components'
      }),
      timestamp: new Date()
    },
    {
      type: 'tool_result' as const,
      id: 'search-1',
      content: 'Found 15 React component files:\n- App.tsx\n- MessageEvents.tsx\n- ToolEvents.tsx\n- ConnectedToolChip.tsx\n- StackedToolEvents.tsx\n- CollapsibleContent.tsx\n- TestStackedUI.tsx\n- Modal.tsx\n- Sidebar.tsx\n- DarkModeToggle.tsx\n- ErrorBoundary/index.tsx\n- LoadingSkeleton/index.tsx\n- Toast/index.tsx\n- MessageActionButtons.tsx\n- ui/button.tsx',
      success: true,
      timestamp: new Date()
    }
  ];

  const mockToolEvents2 = [
    {
      type: 'tool_call' as const,
      name: 'read_file',
      args: JSON.stringify({ 
        path: '/src/components/App.tsx'
      }),
      timestamp: new Date()
    },
    {
      type: 'tool_result' as const,
      id: 'read-1',
      content: 'import { useState, useEffect, useCallback, memo, Suspense, useRef } from "react";\nimport { useSessions } from "./hooks/useSessions";\n// ... rest of App.tsx content ...',
      success: true,
      timestamp: new Date()
    }
  ];

  const mockToolEvents3 = [
    {
      type: 'tool_call' as const,
      name: 'edit_file',
      args: JSON.stringify({ 
        path: '/src/components/MessageEvents.tsx',
        changes: [
          { line: 45, content: 'Added CollapsibleContent import' }
        ]
      }),
      timestamp: new Date()
    },
    {
      type: 'tool_result' as const,
      id: 'edit-1',
      content: 'Successfully updated MessageEvents.tsx with CollapsibleContent integration.',
      success: true,
      timestamp: new Date()
    }
  ];

  // Mock message content for testing text collapse
  const longMessageContent = `This is a long message to test the text collapse functionality. 

Here's line 1 of the content that should be visible.
Here's line 2 of the content that should be visible.
Here's line 3 of the content that should be visible.  
Here's line 4 of the content that should be visible.
Here's line 5 of the content that should be visible.

This is line 6 and beyond that should be collapsed by default in non-latest messages.
This is line 7 with additional content.
This is line 8 with even more content.
This is line 9 that continues the message.
This is line 10 with the final content of this long message.

## Code Example

\`\`\`tsx
const MyComponent: React.FC = () => {
  const [isExpanded, setIsExpanded] = useState(false);
  
  return (
    <div className="component">
      <h1>Test Component</h1>
      <p>This is a code example that should also be collapsible</p>
    </div>
  );
};
\`\`\`

And here's some more text after the code block to make this message even longer and test the collapse functionality thoroughly.`;

  return (
    <div className="max-w-4xl mx-auto p-4 space-y-6">
      <div className="bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
        <h2 className="text-lg font-semibold mb-2 text-blue-800 dark:text-blue-200">
          🧪 Testing New Stacked UI Features
        </h2>
        <p className="text-sm text-blue-700 dark:text-blue-300">
          This demo shows the new stacked tool calls and message text collapse functionality.
        </p>
      </div>

      {/* Test Stacked Tool Events */}
      <div className="space-y-4">
        <h3 className="text-md font-semibold">Stacked Tool Events (Multiple Calls)</h3>
        <p className="text-sm text-muted-foreground">
          Multiple tool calls should be stacked with only the latest visible by default:
        </p>
        
        {/* This should show as a stack with only the latest (edit_file) visible */}
        <StackedToolEvents
          events={[...mockToolEvents, ...mockToolEvents2, ...mockToolEvents3]}
          className="mx-2 lg:mx-3"
          maxVisible={1}
        />
      </div>

      {/* Test Individual Tool Events */}
      <div className="space-y-4">
        <h3 className="text-md font-semibold">Single Tool Event</h3>
        <p className="text-sm text-muted-foreground">
          Single tool call should show normally:
        </p>
        
        <StackedToolEvents
          events={mockToolEvents}
          className="mx-2 lg:mx-3"
          maxVisible={1}
        />
      </div>

      {/* Test Message Text Collapse */}
      <div className="space-y-4">
        <h3 className="text-md font-semibold">Message Text Collapse</h3>
        
        <div className="space-y-3">
          <h4 className="text-sm font-medium">Latest Message (should show full content):</h4>
          <MessageEvent
            role="assistant"
            agent="Test Agent"
            content={longMessageContent}
            isLatest={true}
          />
        </div>

        <div className="space-y-3">
          <h4 className="text-sm font-medium">Non-Latest Message (should be collapsed to 5 lines):</h4>
          <MessageEvent
            role="assistant"
            agent="Test Agent"
            content={longMessageContent}
            isLatest={false}
          />
        </div>

        <div className="space-y-3">
          <h4 className="text-sm font-medium">Short Message (shouldn't collapse):</h4>
          <MessageEvent
            role="user"
            agent={null}
            content="This is a short message that shouldn't need collapsing."
            isLatest={false}
          />
        </div>
      </div>

      {/* Test CollapsibleContent directly */}
      <div className="space-y-4">
        <h3 className="text-md font-semibold">Direct CollapsibleContent Test</h3>
        
        <div className="border rounded-lg p-4">
          <h4 className="text-sm font-medium mb-2">Collapsible Content (5 lines max):</h4>
          <CollapsibleContent maxLines={5} isLatest={false}>
            <div className="prose text-sm">
              <p>Line 1: This is the first line of collapsible content.</p>
              <p>Line 2: This is the second line that should be visible.</p>
              <p>Line 3: This is the third line in the content.</p>
              <p>Line 4: This is the fourth line of the test content.</p>
              <p>Line 5: This is the fifth and last visible line.</p>
              <p>Line 6: This line should be hidden by default.</p>
              <p>Line 7: This line should also be hidden.</p>
              <p>Line 8: Another hidden line for testing.</p>
              <p>Line 9: More hidden content here.</p>
              <p>Line 10: Final hidden line of content.</p>
            </div>
          </CollapsibleContent>
        </div>

        <div className="border rounded-lg p-4">
          <h4 className="text-sm font-medium mb-2">Non-Collapsible Content (isLatest=true):</h4>
          <CollapsibleContent maxLines={5} isLatest={true}>
            <div className="prose text-sm">
              <p>Line 1: This content should show in full.</p>
              <p>Line 2: Because isLatest is true.</p>
              <p>Line 3: All lines should be visible.</p>
              <p>Line 4: No collapse button should appear.</p>
              <p>Line 5: This is the fifth line.</p>
              <p>Line 6: This line should still be visible.</p>
              <p>Line 7: And this one too.</p>
              <p>Line 8: All content remains expanded.</p>
            </div>
          </CollapsibleContent>
        </div>
      </div>

      <div className="bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg p-4">
        <h4 className="text-sm font-semibold mb-2 text-green-800 dark:text-green-200">
          ✅ Expected Behavior
        </h4>
        <ul className="text-xs text-green-700 dark:text-green-300 space-y-1">
          <li>• Multiple tool calls should stack with "Show X more" button</li>
          <li>• Latest messages should show full content</li>
          <li>• Non-latest messages should collapse to 5 lines with expand button</li>
          <li>• Short messages should not show collapse functionality</li>
          <li>• All interactions should be smooth with animations</li>
        </ul>
      </div>
    </div>
  );
};

export default TestStackedUI;