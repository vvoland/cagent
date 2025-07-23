import { useState } from 'react';
import { ToolChip, ToolChipGroup } from '../index';
import type { ChipState, ChipStatus } from '../types';

// Demo component to showcase the tool chip functionality
export const ToolChipDemo = () => {
  const [chipStates, setChipStates] = useState<Record<string, ChipState>>({});

  const handleStateChange = (chipId: string, newState: ChipState) => {
    setChipStates(prev => ({ ...prev, [chipId]: newState }));
  };

  const sampleToolCalls = [
    {
      id: 'call_1',
      name: 'read_file',
      args: JSON.stringify({ path: '/Users/example/document.txt', encoding: 'utf-8' }),
      status: 'success' as ChipStatus
    },
    {
      id: 'call_2', 
      name: 'shell_command',
      args: JSON.stringify({ cmd: 'ls -la', cwd: '/Users/example' }),
      status: 'loading' as ChipStatus
    },
    {
      id: 'call_3',
      name: 'search_files',
      args: JSON.stringify({ 
        query: 'TODO',
        path: '/src',
        extensions: ['.js', '.ts', '.jsx', '.tsx']
      }),
      status: 'error' as ChipStatus
    },
    {
      id: 'call_4',
      name: 'web_request',
      args: JSON.stringify({ 
        url: 'https://api.example.com/data',
        method: 'GET',
        headers: { 'Content-Type': 'application/json' }
      }),
      status: 'success' as ChipStatus
    }
  ];

  const sampleToolResults = [
    {
      id: 'result_1',
      name: 'read_file',
      result: 'File contents:\n\nHello World!\nThis is a sample file with some content that demonstrates\nhow the tool chip handles longer text content.',
      status: 'success' as ChipStatus
    },
    {
      id: 'result_2',
      name: 'database_query', 
      result: JSON.stringify({
        data: [
          { id: 1, name: 'John Doe', email: 'john@example.com' },
          { id: 2, name: 'Jane Smith', email: 'jane@example.com' }
        ],
        count: 2,
        success: true
      }, null, 2),
      status: 'success' as ChipStatus
    }
  ];

  return (
    <div className="p-6 space-y-8 bg-background text-foreground">
      <div className="space-y-4">
        <h2 className="text-2xl font-bold">Tool Call Chips Demo</h2>
        <p className="text-muted-foreground">
          Interactive demonstration of the optimized tool call chip system with three states:
          collapsed (60% smaller), preview, and expanded.
        </p>
      </div>

      <div className="space-y-6">
        <div className="space-y-3">
          <h3 className="text-lg font-semibold">Tool Calls</h3>
          <p className="text-sm text-muted-foreground">
            Click chips to cycle through states: collapsed → preview → expanded → collapsed
          </p>
          <ToolChipGroup className="flex-wrap">
            {sampleToolCalls.map((tool) => (
              <ToolChip
                key={tool.id}
                id={tool.id}
                name={tool.name}
                type={null} // Auto-detect from name
                status={tool.status}
                args={tool.args}
                variant="call"
                timestamp={new Date()}
                initialState="collapsed"
                onStateChange={(state) => handleStateChange(tool.id, state)}
                className="m-1"
              />
            ))}
          </ToolChipGroup>
        </div>

        <div className="space-y-3">
          <h3 className="text-lg font-semibold">Tool Results</h3>
          <p className="text-sm text-muted-foreground">
            Results show the output from tool executions with copy functionality
          </p>
          <ToolChipGroup className="flex-wrap">
            {sampleToolResults.map((result) => (
              <ToolChip
                key={result.id}
                id={result.id}
                name={result.name}
                type={null} // Auto-detect from name
                status={result.status}
                result={result.result}
                variant="result"
                timestamp={new Date()}
                initialState="collapsed"
                onStateChange={(state) => handleStateChange(result.id, state)}
                className="m-1"
              />
            ))}
          </ToolChipGroup>
        </div>

        <div className="space-y-3">
          <h3 className="text-lg font-semibold">Features Demonstrated</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-sm">
            <div className="space-y-2">
              <h4 className="font-medium">Core Features:</h4>
              <ul className="space-y-1 text-muted-foreground list-disc list-inside">
                <li>60% size reduction in collapsed state</li>
                <li>Three interactive states with smooth animations</li>
                <li>Tool type auto-detection and icon mapping</li>
                <li>Status indicators (idle, loading, success, error)</li>
                <li>Copy functionality for tool data</li>
              </ul>
            </div>
            <div className="space-y-2">
              <h4 className="font-medium">Accessibility:</h4>
              <ul className="space-y-1 text-muted-foreground list-disc list-inside">
                <li>WCAG 2.1 AA compliant with ARIA labels</li>
                <li>Keyboard navigation (Enter, Space, Arrow keys, Escape)</li>
                <li>Touch-friendly 44px targets on mobile</li>
                <li>Focus indicators and screen reader support</li>
                <li>High contrast mode support</li>
              </ul>
            </div>
          </div>
        </div>

        <div className="space-y-3">
          <h3 className="text-lg font-semibold">Current States</h3>
          <div className="bg-muted/30 p-4 rounded-lg">
            <pre className="text-xs font-mono">
              {JSON.stringify(chipStates, null, 2)}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ToolChipDemo;