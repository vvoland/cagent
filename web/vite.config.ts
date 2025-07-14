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
      "/api/agents": "http://localhost:8080",
      "/api/sessions": "http://localhost:8080",
      "/api/sessions/:id": "http://localhost:8080",
      "/api/sessions/:id/agent/:agent": "http://localhost:8080",
    },
  },
});
