import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "REACT_APP_");

  return {
    plugins: [react()],
    define: {
      "process.env.REACT_APP_API_BASE_URL": JSON.stringify(env.REACT_APP_API_BASE_URL || ""),
    },
    server: { port: 3000, strictPort: true },
    build: { outDir: "build" },
    test: { globals: true, environment: "jsdom" },
  };
});
