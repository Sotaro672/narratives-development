import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import federation from "@module-federation/vite";
import mfConfig from "./module-federation.config";

export default defineConfig({
  plugins: [react(), federation(mfConfig)],
  server: {
    port: 4005,
    open: false,
    cors: true, // shellからのアクセス許可
  },
  build: {
    target: "esnext",
    outDir: "dist",
  },
  resolve: {
    alias: {
      "@": "/src",
    },
  },
});
