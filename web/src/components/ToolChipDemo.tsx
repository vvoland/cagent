import { memo } from 'react';
import { ConnectedToolChipGroup, ConnectedToolChip, useConnectedToolCalls } from './ConnectedToolChip';

interface DemoEvent {
  type: 'tool_call' | 'tool_result';
  name?: string;
  args?: string;
  id?: string;
  content?: string;
  success?: boolean;
  timestamp?: Date;
}

const ToolChipDemo = memo(() => {
  // Sample event sequences to demonstrate the connected tool chip system
  const sampleEvents: DemoEvent[] = [
    // File operations sequence
    {
      type: 'tool_call',
      name: 'read_file',
      args: JSON.stringify({ path: '/src/components/App.tsx' }, null, 2),
      timestamp: new Date('2024-01-01T10:00:00Z')
    },
    {
      type: 'tool_result',
      id: 'read_1',
      content: 'import React from "react";\nimport { useState } from "react";\n\nfunction App() {\n  return <div>Hello World</div>;\n}\n\nexport default App;',
      success: true,
      timestamp: new Date('2024-01-01T10:00:01Z')
    },
    
    // Search operation sequence
    {
      type: 'tool_call',
      name: 'search_files',
      args: JSON.stringify({ pattern: '*.tsx', path: '/src', maxResults: 10 }, null, 2),
      timestamp: new Date('2024-01-01T10:01:00Z')
    },
    {
      type: 'tool_result',
      id: 'search_1',
      content: 'Found 12 files:\n- /src/App.tsx\n- /src/components/Button.tsx\n- /src/components/Modal.tsx\n- /src/hooks/useAuth.tsx\n- /src/pages/Home.tsx\n- /src/pages/Dashboard.tsx\n- /src/utils/helpers.tsx\n- /src/types/index.tsx\n- /src/components/Header.tsx\n- /src/components/Footer.tsx\n- /src/components/Sidebar.tsx\n- /src/components/Loading.tsx',
      success: true,
      timestamp: new Date('2024-01-01T10:01:02Z')
    },
    
    // Database operation with error
    {
      type: 'tool_call',
      name: 'query_database',
      args: JSON.stringify({ query: 'SELECT * FROM users WHERE active = true', limit: 100 }, null, 2),
      timestamp: new Date('2024-01-01T10:02:00Z')
    },
    {
      type: 'tool_result',
      id: 'query_1',
      content: 'Error: Connection refused\nDatabase server is not responding on port 5432\nPlease check your database configuration and ensure the server is running.',
      success: false,
      timestamp: new Date('2024-01-01T10:02:03Z')
    },
    
    // API call sequence
    {
      type: 'tool_call',
      name: 'fetch_api',
      args: JSON.stringify({ url: 'https://api.github.com/user/repos', method: 'GET', headers: { 'Authorization': 'Bearer ***' } }, null, 2),
      timestamp: new Date('2024-01-01T10:03:00Z')
    },
    {
      type: 'tool_result',
      id: 'api_1',
      content: '[\n  {\n    "name": "awesome-project",\n    "full_name": "user/awesome-project",\n    "private": false,\n    "description": "An awesome React application",\n    "stargazers_count": 42,\n    "language": "TypeScript"\n  },\n  {\n    "name": "another-repo",\n    "full_name": "user/another-repo",\n    "private": true,\n    "description": "Private repository",\n    "stargazers_count": 0,\n    "language": "JavaScript"\n  }\n]',
      success: true,
      timestamp: new Date('2024-01-01T10:03:01Z')
    },
    
    // Memory operation
    {
      type: 'tool_call',
      name: 'add_memory',
      args: JSON.stringify({ memory: 'User prefers TypeScript over JavaScript for new projects' }, null, 2),
      timestamp: new Date('2024-01-01T10:04:00Z')
    },
    {
      type: 'tool_result',
      id: 'memory_1',
      content: 'Memory added successfully with ID: mem_12345',
      success: true,
      timestamp: new Date('2024-01-01T10:04:01Z')
    },
    
    // Shell command
    {
      type: 'tool_call',
      name: 'execute_shell',
      args: JSON.stringify({ command: 'npm install react@latest', workingDir: '/project' }, null, 2),
      timestamp: new Date('2024-01-01T10:05:00Z')
    },
    {
      type: 'tool_result',
      id: 'shell_1',
      content: '+ react@18.2.0\nadded 3 packages, and audited 1425 packages in 2s\n\n85 packages are looking for funding\n  run `npm fund` for details\n\nfound 0 vulnerabilities',
      success: true,
      timestamp: new Date('2024-01-01T10:05:05Z')
    },
    
    // Pending operation (no result yet)
    {
      type: 'tool_call',
      name: 'analyze_code',
      args: JSON.stringify({ path: '/src', metrics: ['complexity', 'coverage', 'performance'] }, null, 2),
      timestamp: new Date('2024-01-01T10:06:00Z')
    }
  ];

  const { connectedToolCalls, toggleExpanded, isExpanded } = useConnectedToolCalls(sampleEvents);

  return (
    <div className="max-w-4xl mx-auto p-6 space-y-8">
      <div className="text-center space-y-2">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
          Connected Tool Chip Demo
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Interactive demonstration of the connected tool call UI system
        </p>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
        <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
          Connected Tool Operations
        </h2>
        <p className="text-gray-600 dark:text-gray-400 mb-6">
          Each operation shows the tool call and its result as a connected unit. Click to expand and see details.
        </p>
        
        <ConnectedToolChipGroup className="space-y-3">
          {connectedToolCalls.map((toolCall) => (
            <ConnectedToolChip
              key={toolCall.id}
              toolCall={toolCall}
              onToggle={toggleExpanded}
              expanded={isExpanded(toolCall.id)}
              className="transition-all hover:shadow-md hover:scale-[1.005] active:scale-100"
            />
          ))}
        </ConnectedToolChipGroup>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-lg font-semibold mb-3 text-gray-900 dark:text-white">
            Features
          </h3>
          <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-green-500 rounded-full"></span>
              Visual connection between calls and results
            </li>
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-blue-500 rounded-full"></span>
              Status indicators (success, error, pending)
            </li>
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-purple-500 rounded-full"></span>
              Tool type auto-detection and color coding
            </li>
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-orange-500 rounded-full"></span>
              Expandable content with copy functionality
            </li>
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-teal-500 rounded-full"></span>
              Mobile-responsive design
            </li>
            <li className="flex items-center gap-2">
              <span className="w-2 h-2 bg-pink-500 rounded-full"></span>
              Keyboard navigation support
            </li>
          </ul>
        </div>

        <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
          <h3 className="text-lg font-semibold mb-3 text-gray-900 dark:text-white">
            Tool Types
          </h3>
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="flex items-center gap-2 p-2 bg-blue-50 dark:bg-blue-950/30 rounded border border-blue-200 dark:border-blue-800">
              <span className="w-2 h-2 bg-blue-500 rounded-full"></span>
              <span className="text-blue-800 dark:text-blue-200">File Operations</span>
            </div>
            <div className="flex items-center gap-2 p-2 bg-purple-50 dark:bg-purple-950/30 rounded border border-purple-200 dark:border-purple-800">
              <span className="w-2 h-2 bg-purple-500 rounded-full"></span>
              <span className="text-purple-800 dark:text-purple-200">Search</span>
            </div>
            <div className="flex items-center gap-2 p-2 bg-teal-50 dark:bg-teal-950/30 rounded border border-teal-200 dark:border-teal-800">
              <span className="w-2 h-2 bg-teal-500 rounded-full"></span>
              <span className="text-teal-800 dark:text-teal-200">Database</span>
            </div>
            <div className="flex items-center gap-2 p-2 bg-yellow-50 dark:bg-yellow-950/30 rounded border border-yellow-200 dark:border-yellow-800">
              <span className="w-2 h-2 bg-yellow-500 rounded-full"></span>
              <span className="text-yellow-800 dark:text-yellow-200">API Calls</span>
            </div>
            <div className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-gray-950/30 rounded border border-gray-200 dark:border-gray-800">
              <span className="w-2 h-2 bg-gray-500 rounded-full"></span>
              <span className="text-gray-800 dark:text-gray-200">Shell</span>
            </div>
            <div className="flex items-center gap-2 p-2 bg-pink-50 dark:bg-pink-950/30 rounded border border-pink-200 dark:border-pink-800">
              <span className="w-2 h-2 bg-pink-500 rounded-full"></span>
              <span className="text-pink-800 dark:text-pink-200">Memory</span>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-900 rounded-lg border border-gray-200 dark:border-gray-800 p-6">
        <h3 className="text-lg font-semibold mb-3 text-gray-900 dark:text-white">
          Usage Instructions
        </h3>
        <div className="space-y-3 text-sm text-gray-600 dark:text-gray-400">
          <p>
            <strong>Click</strong> any tool operation above to expand and see the full details of both the call and result.
          </p>
          <p>
            <strong>Copy</strong> buttons appear when expanded to copy the call parameters or result content.
          </p>
          <p>
            <strong>Status indicators</strong> show at a glance whether operations succeeded, failed, or are still pending.
          </p>
          <p>
            <strong>Color coding</strong> helps identify different types of operations quickly.
          </p>
        </div>
      </div>
    </div>
  );
});

ToolChipDemo.displayName = 'ToolChipDemo';

export default ToolChipDemo;