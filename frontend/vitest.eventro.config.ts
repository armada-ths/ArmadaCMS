import { defineConfig, mergeConfig } from "vitest/config";
import browserConfig from "./vitest.photos.config.ts";

export default mergeConfig(
  browserConfig,
  defineConfig({
    test: { include: ["src/components/Dashboard/*.browser.tsx"] },
  }),
);
