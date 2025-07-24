import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    react({
      // Use SWC for better performance
      jsxRuntime: 'automatic',
    }), 
    tailwindcss()
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    // Optimize build performance
    target: 'es2020',
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
      },
    },
    // Enable code splitting
    rollupOptions: {
      output: {
        manualChunks: {
          // Vendor chunks for better caching
          'react-vendor': ['react', 'react-dom'],
          'ui-vendor': ['@radix-ui/react-select', '@radix-ui/react-slot', 'lucide-react'],
          'markdown-vendor': ['react-markdown', 'react-syntax-highlighter', 'remark-gfm'],
          'utils': ['class-variance-authority', 'clsx', 'tailwind-merge'],
        },
      },
    },
    // Optimize chunk size warnings
    chunkSizeWarningLimit: 1000,
    // Enable sourcemaps for debugging
    sourcemap: false,
  },
  server: {
    // Development server optimizations
    host: true,
    port: 5173,
    strictPort: true,
    // Hot reload optimizations
    hmr: {
      overlay: true,
    },
    proxy: {
      "/api/agents": "http://localhost:8080",
      "/api/sessions": "http://localhost:8080",
      "/api/sessions/:id": "http://localhost:8080",
      "/api/sessions/:id/agent/:agent": "http://localhost:8080",
    },
  },
  preview: {
    port: 4173,
    strictPort: true,
  },
  // Performance optimizations
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      'react-markdown',
      'react-syntax-highlighter',
      'remark-gfm',
      '@radix-ui/react-select',
      '@radix-ui/react-slot',
      'lucide-react',
      'class-variance-authority',
      'clsx',
      'tailwind-merge'
    ],
  },
});