import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// Dev server proxies the API so the browser talks to one origin.
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8000",
      "/health": "http://127.0.0.1:8000",
    },
  },
});
