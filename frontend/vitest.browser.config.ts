import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";
import { fileURLToPath } from "node:url";
import { playwright } from "@vitest/browser-playwright";

export default defineConfig({
  plugins: [react()],
  optimizeDeps: { include: ["react-admin", "react-dom/client"] },
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  test: {
    include: ["src/**/*.browser.tsx"],
    testTimeout: 15000,
    browser: {
      enabled: true,
      headless: true,
      provider: playwright(),
      instances: [{ browser: "chromium" }],
    },
  },
});
