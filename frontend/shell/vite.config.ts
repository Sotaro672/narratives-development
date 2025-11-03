import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import federation from "@module-federation/vite";
import mfConfig from "./module-federation.config";

export default defineConfig({
  plugins: [
    react(),
    federation(mfConfig), // ← federation設定を読み込み
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
