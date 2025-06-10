import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/agents": "http://localhost:8080",
      "/sessions": "http://localhost:8080",
      "/sessions/:id": "http://localhost:8080",
      "/sessions/:id/agent/:agent": "http://localhost:8080",
    },
  },
});
