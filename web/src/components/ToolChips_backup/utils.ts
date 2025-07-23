import { 
  FileText, 
  Search, 
  Terminal, 
  Globe, 
  Database, 
  Zap, 
  BarChart3, 
  Brain,
  Settings,
  CheckCircle,
  AlertCircle,
  Clock,
  Loader2
} from 'lucide-react';

// Move type definitions here to avoid cross-file import issues
export type ToolType = 
  | 'file' 
  | 'search' 
  | 'shell' 
  | 'web' 
  | 'database' 
  | 'api' 
  | 'analysis' 
  | 'memory'
  | 'default';

export const getToolIcon = (toolType: ToolType) => {
  switch (toolType) {
    case 'file':
      return FileText;
    case 'search':
      return Search;
    case 'shell':
      return Terminal;
    case 'web':
      return Globe;
    case 'database':
      return Database;
    case 'api':
      return Zap;
    case 'analysis':
      return BarChart3;
    case 'memory':
      return Brain;
    default:
      return Settings;
  }
};

export const getStatusIcon = (status: 'idle' | 'loading' | 'success' | 'error') => {
  switch (status) {
    case 'loading':
      return Loader2;
    case 'success':
      return CheckCircle;
    case 'error':
      return AlertCircle;
    default:
      return Clock;
  }
};

export const getToolTypeFromName = (toolName: string): ToolType => {
  const name = toolName.toLowerCase();
  
  if (name.includes('file') || name.includes('read') || name.includes('write') || name.includes('edit')) {
    return 'file';
  }
  if (name.includes('search') || name.includes('find') || name.includes('grep')) {
    return 'search';
  }
  if (name.includes('shell') || name.includes('bash') || name.includes('cmd') || name.includes('exec')) {
    return 'shell';
  }
  if (name.includes('web') || name.includes('http') || name.includes('fetch') || name.includes('url')) {
    return 'web';
  }
  if (name.includes('database') || name.includes('db') || name.includes('sql') || name.includes('query')) {
    return 'database';
  }
  if (name.includes('api') || name.includes('request') || name.includes('post') || name.includes('get')) {
    return 'api';
  }
  if (name.includes('analyze') || name.includes('analysis') || name.includes('chart') || name.includes('report')) {
    return 'analysis';
  }
  if (name.includes('memory') || name.includes('remember') || name.includes('store') || name.includes('recall')) {
    return 'memory';
  }
  
  return 'default';
};

export const getChipTheme = (toolType: ToolType, variant: 'call' | 'result') => {
  const themes = {
    file: {
      call: {
        bg: 'bg-blue-50 dark:bg-blue-950/30',
        border: 'border-blue-200 dark:border-blue-800',
        text: 'text-blue-800 dark:text-blue-200',
        icon: 'text-blue-600 dark:text-blue-400',
        hover: 'hover:bg-blue-100 dark:hover:bg-blue-900/40'
      },
      result: {
        bg: 'bg-blue-100 dark:bg-blue-900/50',
        border: 'border-blue-300 dark:border-blue-700',
        text: 'text-blue-900 dark:text-blue-100',
        icon: 'text-blue-700 dark:text-blue-300',
        hover: 'hover:bg-blue-200 dark:hover:bg-blue-800/60'
      }
    },
    search: {
      call: {
        bg: 'bg-purple-50 dark:bg-purple-950/30',
        border: 'border-purple-200 dark:border-purple-800',
        text: 'text-purple-800 dark:text-purple-200',
        icon: 'text-purple-600 dark:text-purple-400',
        hover: 'hover:bg-purple-100 dark:hover:bg-purple-900/40'
      },
      result: {
        bg: 'bg-purple-100 dark:bg-purple-900/50',
        border: 'border-purple-300 dark:border-purple-700',
        text: 'text-purple-900 dark:text-purple-100',
        icon: 'text-purple-700 dark:text-purple-300',
        hover: 'hover:bg-purple-200 dark:hover:bg-purple-800/60'
      }
    },
    shell: {
      call: {
        bg: 'bg-gray-50 dark:bg-gray-950/30',
        border: 'border-gray-200 dark:border-gray-800',
        text: 'text-gray-800 dark:text-gray-200',
        icon: 'text-gray-600 dark:text-gray-400',
        hover: 'hover:bg-gray-100 dark:hover:bg-gray-900/40'
      },
      result: {
        bg: 'bg-gray-100 dark:bg-gray-900/50',
        border: 'border-gray-300 dark:border-gray-700',
        text: 'text-gray-900 dark:text-gray-100',
        icon: 'text-gray-700 dark:text-gray-300',
        hover: 'hover:bg-gray-200 dark:hover:bg-gray-800/60'
      }
    },
    web: {
      call: {
        bg: 'bg-indigo-50 dark:bg-indigo-950/30',
        border: 'border-indigo-200 dark:border-indigo-800',
        text: 'text-indigo-800 dark:text-indigo-200',
        icon: 'text-indigo-600 dark:text-indigo-400',
        hover: 'hover:bg-indigo-100 dark:hover:bg-indigo-900/40'
      },
      result: {
        bg: 'bg-indigo-100 dark:bg-indigo-900/50',
        border: 'border-indigo-300 dark:border-indigo-700',
        text: 'text-indigo-900 dark:text-indigo-100',
        icon: 'text-indigo-700 dark:text-indigo-300',
        hover: 'hover:bg-indigo-200 dark:hover:bg-indigo-800/60'
      }
    },
    database: {
      call: {
        bg: 'bg-teal-50 dark:bg-teal-950/30',
        border: 'border-teal-200 dark:border-teal-800',
        text: 'text-teal-800 dark:text-teal-200',
        icon: 'text-teal-600 dark:text-teal-400',
        hover: 'hover:bg-teal-100 dark:hover:bg-teal-900/40'
      },
      result: {
        bg: 'bg-teal-100 dark:bg-teal-900/50',
        border: 'border-teal-300 dark:border-teal-700',
        text: 'text-teal-900 dark:text-teal-100',
        icon: 'text-teal-700 dark:text-teal-300',
        hover: 'hover:bg-teal-200 dark:hover:bg-teal-800/60'
      }
    },
    api: {
      call: {
        bg: 'bg-yellow-50 dark:bg-yellow-950/30',
        border: 'border-yellow-200 dark:border-yellow-800',
        text: 'text-yellow-800 dark:text-yellow-200',
        icon: 'text-yellow-600 dark:text-yellow-400',
        hover: 'hover:bg-yellow-100 dark:hover:bg-yellow-900/40'
      },
      result: {
        bg: 'bg-yellow-100 dark:bg-yellow-900/50',
        border: 'border-yellow-300 dark:border-yellow-700',
        text: 'text-yellow-900 dark:text-yellow-100',
        icon: 'text-yellow-700 dark:text-yellow-300',
        hover: 'hover:bg-yellow-200 dark:hover:bg-yellow-800/60'
      }
    },
    analysis: {
      call: {
        bg: 'bg-green-50 dark:bg-green-950/30',
        border: 'border-green-200 dark:border-green-800',
        text: 'text-green-800 dark:text-green-200',
        icon: 'text-green-600 dark:text-green-400',
        hover: 'hover:bg-green-100 dark:hover:bg-green-900/40'
      },
      result: {
        bg: 'bg-green-100 dark:bg-green-900/50',
        border: 'border-green-300 dark:border-green-700',
        text: 'text-green-900 dark:text-green-100',
        icon: 'text-green-700 dark:text-green-300',
        hover: 'hover:bg-green-200 dark:hover:bg-green-800/60'
      }
    },
    memory: {
      call: {
        bg: 'bg-pink-50 dark:bg-pink-950/30',
        border: 'border-pink-200 dark:border-pink-800',
        text: 'text-pink-800 dark:text-pink-200',
        icon: 'text-pink-600 dark:text-pink-400',
        hover: 'hover:bg-pink-100 dark:hover:bg-pink-900/40'
      },
      result: {
        bg: 'bg-pink-100 dark:bg-pink-900/50',
        border: 'border-pink-300 dark:border-pink-700',
        text: 'text-pink-900 dark:text-pink-100',
        icon: 'text-pink-700 dark:text-pink-300',
        hover: 'hover:bg-pink-200 dark:hover:bg-pink-800/60'
      }
    },
    default: {
      call: {
        bg: 'bg-slate-50 dark:bg-slate-950/30',
        border: 'border-slate-200 dark:border-slate-800',
        text: 'text-slate-800 dark:text-slate-200',
        icon: 'text-slate-600 dark:text-slate-400',
        hover: 'hover:bg-slate-100 dark:hover:bg-slate-900/40'
      },
      result: {
        bg: 'bg-slate-100 dark:bg-slate-900/50',
        border: 'border-slate-300 dark:border-slate-700',
        text: 'text-slate-900 dark:text-slate-100',
        icon: 'text-slate-700 dark:text-slate-300',
        hover: 'hover:bg-slate-200 dark:hover:bg-slate-800/60'
      }
    }
  };

  return themes[toolType]?.[variant] || themes.default[variant];
};