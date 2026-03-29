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

  // ✅ Adds @ alias and keeps your React Admin debug aliases
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
      ...getAliasesToDebugInProduction(),
    },
  },

  base: "./",
}));

function getAliasesToDebugInProduction() {
  return {
    "react-admin": path.resolve(__dirname, "./node_modules/react-admin/src"),
    "ra-core": path.resolve(__dirname, "./node_modules/ra-core/src"),
    "ra-ui-materialui": path.resolve(
      __dirname,
      "./node_modules/ra-ui-materialui/src",
    ),
    "ra-i18n-polyglot": path.resolve(
      __dirname,
      "./node_modules/ra-i18n-polyglot/src",
    ),
    "ra-language-english": path.resolve(
      __dirname,
      "./node_modules/ra-language-english/src",
    ),
    "ra-data-json-server": path.resolve(
      __dirname,
      "./node_modules/ra-data-json-server/src",
    ),
    "ra-data-simple-rest": path.resolve(
      __dirname,
      "./node_modules/ra-data-simple-rest/src",
    ),
    "ra-data-fakerest": path.resolve(
      __dirname,
      "./node_modules/ra-data-fakerest/src",
    ),
  };
}
