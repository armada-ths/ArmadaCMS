import { defineConfig, mergeConfig } from "vitest/config";
import { playwright } from "@vitest/browser-playwright";
import viteConfig from "./vite.config";

export default defineConfig((environment) =>
  mergeConfig(viteConfig(environment), {
    test: {
      include: [
        "src/components/LoungeItem/*.browser.tsx",
        "src/components/LoungeOrder/*.browser.tsx",
      ],
      testTimeout: 15000,
      browser: {
        enabled: true,
        headless: true,
        provider: playwright(),
        instances: [{ browser: "chromium" }],
      },
    },
  }),
);
