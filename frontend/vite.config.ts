import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [react()],

  server: {
    host: true,
    watch: {
      // Required for HMR with Docker volume mounts on Windows/macOS
      usePolling: true,
      interval: 1000,
    },
  },

  build: {
    sourcemap: mode === "development",
    chunkSizeWarningLimit: 850,
    rollupOptions: {
      output: {
        manualChunks(id) {
          const isInPackage = (pkg: string) =>
            id.includes(`/node_modules/${pkg}/`) ||
            id.includes(`\\node_modules\\${pkg}\\`);

          if (
            ["react", "react-dom", "react-router", "react-router-dom"].some(
              isInPackage,
            )
          ) {
            return "vendor-react";
          }

          if (
            [
              "@mui/material",
              "@mui/icons-material",
              "@emotion/react",
              "@emotion/styled",
              "react-admin",
              "ra-core",
              "ra-ui-materialui",
              "ra-data-json-server",
              "ra-data-simple-rest",
            ].some(isInPackage)
          ) {
            return "vendor-admin";
          }

          return undefined;
        },
      },
    },
  },

  // ✅ Adds @ alias and deduplicates shared packages.
  // dedupe ensures pnpm's strict isolation never produces two separate
  // module instances of react-router (which would break react-admin's
  // router context on Linux where pnpm uses real symlinks).
  resolve: {
    dedupe: [
      "react",
      "react-dom",
      "react-router",
      "react-router-dom",
      "react-hook-form",
    ],
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },

  base: "./",
}));
