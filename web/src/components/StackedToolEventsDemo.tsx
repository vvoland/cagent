import { useState } from 'react';
import { StackedToolEvents } from './StackedToolEvents';

// Demo component to showcase the stacked tool calling UI
export const StackedToolEventsDemo = () => {
  const [demoEvents] = useState([
    // First tool call and result
    { type: 'tool_call' as const, name: 'read_file', args: '{"path": "/home/user/document.txt"}', timestamp: new Date() },
    { type: 'tool_result' as const, id: 'result-1', content: 'File content: Hello world!', success: true, timestamp: new Date() },
    
    // Second tool call and result
    { type: 'tool_call' as const, name: 'search_files', args: '{"pattern": "*.js", "path": "/src"}', timestamp: new Date() },
    { type: 'tool_result' as const, id: 'result-2', content: 'Found 15 files:\nindex.js\ncomponent.js\nutil.js\n...', success: true, timestamp: new Date() },
    
    // Third tool call and result
    { type: 'tool_call' as const, name: 'shell', args: '{"command": "npm test"}', timestamp: new Date() },
    { type: 'tool_result' as const, id: 'result-3', content: 'All tests passed!\n✓ 24 tests\n✓ 100% coverage', success: true, timestamp: new Date() },
    
    // Fourth tool call (error case)
    { type: 'tool_call' as const, name: 'web_request', args: '{"url": "https://api.example.com/data"}', timestamp: new Date() },
    { type: 'tool_result' as const, id: 'result-4', content: 'Error: Connection timeout after 30s', success: false, timestamp: new Date() },
  ]);

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-8">
      <div className="text-center">
        <h1 className="text-2xl font-bold mb-2">Stacked Tool Events Demo</h1>
        <p className="text-gray-600 dark:text-gray-400">
          Demonstrates the new stacked tool calling UI with depth effects
        </p>
      </div>

      <div className="space-y-6">
        <div>
          <h2 className="text-lg font-semibold mb-3">Default Stacked View (maxVisible=1)</h2>
          <div className="border rounded-lg p-4 bg-gray-50 dark:bg-gray-900">
            <StackedToolEvents events={demoEvents} maxVisible={1} />
          </div>
          <p className="text-sm text-gray-500 mt-2">
            Shows only the latest tool call by default. Click "Show X more" to expand the stack.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-3">Show More Tools (maxVisible=2)</h2>
          <div className="border rounded-lg p-4 bg-gray-50 dark:bg-gray-900">
            <StackedToolEvents events={demoEvents} maxVisible={2} />
          </div>
          <p className="text-sm text-gray-500 mt-2">
            Shows the latest 2 tool calls with stacking effects on older tools.
          </p>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-3">Features Demonstrated</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <h3 className="font-medium">Visual Effects:</h3>
              <ul className="list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400">
                <li>Latest tool: Full visibility (scale: 1, opacity: 1)</li>
                <li>Second tool: Partial visibility (scale: 0.98, opacity: 0.8, clipped 40%)</li>
                <li>Third tool: More partial (scale: 0.96, opacity: 0.6, clipped 60%)</li>
                <li>Fourth+ tools: Heavily clipped (scale: 0.94, opacity: 0.4, clipped 75%)</li>
              </ul>
            </div>
            <div className="space-y-2">
              <h3 className="font-medium">Interactions:</h3>
              <ul className="list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400">
                <li>Expand/collapse stack functionality</li>
                <li>Counter badges showing hidden tool count</li>
                <li>Individual tool chip expand/collapse</li>
                <li>Copy functionality for tool args and results</li>
                <li>Mobile-friendly 44px touch targets</li>
                <li>Smooth CSS transitions (300ms)</li>
              </ul>
            </div>
          </div>
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-3">Accessibility Features</h2>
          <div className="text-sm space-y-2">
            <ul className="list-disc list-inside space-y-1 text-gray-600 dark:text-gray-400">
              <li><strong>ARIA Labels:</strong> Screen reader support with expand/collapse states</li>
              <li><strong>Keyboard Navigation:</strong> Focus indicators and keyboard interaction</li>
              <li><strong>Touch Targets:</strong> Minimum 44px on mobile, 32px on desktop</li>
              <li><strong>Color Contrast:</strong> WCAG 2.1 AA compliant color themes</li>
              <li><strong>Reduced Motion:</strong> Respects user motion preferences</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
};

export default StackedToolEventsDemo;