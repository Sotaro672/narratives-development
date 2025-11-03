import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import federation from "@module-federation/vite";
import mfConfig from "./module-federation.config";

export default defineConfig({
  plugins: [react(), federation(mfConfig)],
  server: {
    port: 4002,
    open: false,
  },
  build: {
    target: "esnext",
    outDir: "dist",
  },
});
