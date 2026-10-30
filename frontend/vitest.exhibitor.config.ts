import { defineConfig } from "vitest/config";
import browserConfig from "./vitest.photos.config.ts";
import path from "node:path";

export default defineConfig({
  ...browserConfig,
  resolve: { alias: { "@": path.resolve(import.meta.dirname, "src") } },
  test: {
    ...browserConfig.test,
    include: ["src/components/Exhibitor/*.browser.tsx"],
  },
});
