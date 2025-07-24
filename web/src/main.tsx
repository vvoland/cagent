import { StrictMode, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { ErrorBoundary } from './components/ErrorBoundary'
import { ToastProvider } from './components/Toast'
import { useLogger } from './utils/logger'

// Enhanced loading fallback component
const AppFallback = () => {
  const logger = useLogger('AppFallback');
  
  // Log loading state
  logger.info('Application loading...');
  
  return (
    <div className="min-h-screen flex items-center justify-center bg-background text-foreground">
      <div className="flex flex-col items-center gap-4">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary"></div>
        <p className="text-muted-foreground animate-pulse">Loading application...</p>
        <div className="w-32 h-1 bg-muted rounded-full overflow-hidden">
          <div className="h-full bg-primary rounded-full animate-pulse" style={{ width: '60%' }}></div>
        </div>
      </div>
    </div>
  );
};

const rootElement = document.getElementById('root');
if (!rootElement) {
  throw new Error('Root element not found. Please ensure there is a div with id="root" in your HTML.');
}

// Enhanced root rendering with better error handling and providers
const root = createRoot(rootElement);

root.render(
  <StrictMode>
    <ErrorBoundary>
      <ToastProvider>
        <Suspense fallback={<AppFallback />}>
          <App />
        </Suspense>
      </ToastProvider>
    </ErrorBoundary>
  </StrictMode>
);

// Service worker registration for PWA capabilities
if ('serviceWorker' in navigator && process.env.NODE_ENV === 'production') {
  window.addEventListener('load', async () => {
    try {
      const registration = await navigator.serviceWorker.register('/sw.js');
      console.log('SW registered successfully:', registration);
    } catch (registrationError) {
      console.warn('SW registration failed:', registrationError);
    }
  });
}

// Performance monitoring
if (process.env.NODE_ENV === 'development') {
  // Monitor performance metrics
  if ('PerformanceObserver' in window) {
    const observer = new PerformanceObserver((list) => {
      list.getEntries().forEach((entry) => {
        if (entry.entryType === 'paint') {
          console.log(`${entry.name}: ${entry.startTime}ms`);
        }
      });
    });
    
    observer.observe({ entryTypes: ['paint', 'navigation'] });
  }
}