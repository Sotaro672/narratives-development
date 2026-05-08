import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import mf from "./module-federation.config";

export default defineConfig({
  plugins: [react(), mf],
  server: { port: 4002 },
  build: { target: "esnext" },
});
