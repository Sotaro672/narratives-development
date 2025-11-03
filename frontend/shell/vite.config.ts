// frontend/shell/vite.config.ts
import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import { federation } from "@module-federation/vite";
import mfOptions from "./module-federation.config";

export default defineConfig({
  plugins: [
    react(),
    federation(mfOptions) as unknown as PluginOption, // ← options を渡す
  ],
  server: {
    port: 4000,
    open: true,
  },
  build: {
    target: "esnext",
    modulePreload: false,
    outDir: "dist",
  },
});
