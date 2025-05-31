import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/agents": "http://localhost:8080",
      "/sessions": "http://localhost:8080",
      "/sessions/:id": "http://localhost:8080",
      "/sessions/:id/agent/:agent": "http://localhost:8080",
    },
  },
});
