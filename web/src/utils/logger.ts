import { useMemo } from 'react';

// Enhanced logging utility with different levels and structured output

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
}

interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  data?: any;
  component?: string | null;
  userId?: string;
  sessionId?: string;
}

class Logger {
  private level: LogLevel;
  private logs: LogEntry[] = [];
  private maxLogs = 1000;

  constructor(level: LogLevel = LogLevel.INFO) {
    this.level = level;
  }

  private shouldLog(level: LogLevel): boolean {
    return level >= this.level;
  }

  private formatMessage(level: LogLevel, message: string, data?: any, context?: string): void {
    if (!this.shouldLog(level)) return;

    const timestamp = new Date().toISOString();
    const logEntry: LogEntry = {
      timestamp,
      level,
      message,
      data,
      component: context || null,
    };

    // Store log entry
    this.logs.push(logEntry);
    if (this.logs.length > this.maxLogs) {
      this.logs = this.logs.slice(-this.maxLogs);
    }

    // Console output with styling
    const levelName = LogLevel[level];
    const prefix = `[${timestamp}] [${levelName}]${context ? ` [${context}]` : ''}`;
    
    const styles = {
      [LogLevel.DEBUG]: 'color: #6b7280; font-weight: normal;',
      [LogLevel.INFO]: 'color: #2563eb; font-weight: normal;',
      [LogLevel.WARN]: 'color: #d97706; font-weight: bold;',
      [LogLevel.ERROR]: 'color: #dc2626; font-weight: bold;',
    };

    if (data) {
      console.groupCollapsed(`%c${prefix} ${message}`, styles[level]);
      console.log('Data:', data);
      console.groupEnd();
    } else {
      console.log(`%c${prefix} ${message}`, styles[level]);
    }
  }

  debug(message: string, data?: any, context?: string): void {
    this.formatMessage(LogLevel.DEBUG, message, data, context);
  }

  info(message: string, data?: any, context?: string): void {
    this.formatMessage(LogLevel.INFO, message, data, context);
  }

  warn(message: string, data?: any, context?: string): void {
    this.formatMessage(LogLevel.WARN, message, data, context);
  }

  error(message: string, error?: Error | any, context?: string): void {
    this.formatMessage(LogLevel.ERROR, message, error, context);
    
    // In production, you might want to send errors to a monitoring service
    if (process.env.NODE_ENV === 'production' && error) {
      this.reportError(message, error, context);
    }
  }

  // Performance logging
  time(label: string): void {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.time(label);
    }
  }

  timeEnd(label: string): void {
    if (this.shouldLog(LogLevel.DEBUG)) {
      console.timeEnd(label);
    }
  }

  // Get stored logs
  getLogs(level?: LogLevel): LogEntry[] {
    if (level !== undefined) {
      return this.logs.filter(log => log.level >= level);
    }
    return [...this.logs];
  }

  // Clear stored logs
  clearLogs(): void {
    this.logs = [];
  }

  // Set log level
  setLevel(level: LogLevel): void {
    this.level = level;
  }

  // Export logs as JSON
  exportLogs(): string {
    return JSON.stringify(this.logs, null, 2);
  }

  // Report error to external service (placeholder)
  private reportError(message: string, error: any, context?: string): void {
    // TODO: Implement error reporting to external service
    // Example: Sentry, LogRocket, etc.
    console.warn('Error reporting not implemented:', { message, error, context });
  }

  // Component-specific logger factory
  createComponentLogger(componentName: string) {
    return {
      debug: (message: string, data?: any) => this.debug(message, data, componentName),
      info: (message: string, data?: any) => this.info(message, data, componentName),
      warn: (message: string, data?: any) => this.warn(message, data, componentName),
      error: (message: string, error?: any) => this.error(message, error, componentName),
      time: (label: string) => this.time(`${componentName}:${label}`),
      timeEnd: (label: string) => this.timeEnd(`${componentName}:${label}`),
    };
  }
}

// Create singleton logger instance
const logger = new Logger(
  process.env.NODE_ENV === 'development' ? LogLevel.DEBUG : LogLevel.WARN
);

export { logger, Logger };
export default logger;

// Convenience exports
export const log = {
  debug: logger.debug.bind(logger),
  info: logger.info.bind(logger),
  warn: logger.warn.bind(logger),
  error: logger.error.bind(logger),
  time: logger.time.bind(logger),
  timeEnd: logger.timeEnd.bind(logger),
};

// React hook for component logging
export const useLogger = (componentName: string) => {
  return useMemo(() => logger.createComponentLogger(componentName), [componentName]);
};