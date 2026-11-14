import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

// The dev server proxies the Go server so the browser talks to one origin.
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8610",
      "/health": "http://127.0.0.1:8610",
    },
  },
});
