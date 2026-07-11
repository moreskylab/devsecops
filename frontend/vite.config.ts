import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 8080,
    proxy: {
      "/items": {
        target: "http://localhost:8000",
        changeOrigin: true,
      },
      "/v1": {
        target: "http://localhost:4318",
        changeOrigin: true,
      },
    },
  },
  build: {
    target: "es2023",
    sourcemap: true,
  },
});
