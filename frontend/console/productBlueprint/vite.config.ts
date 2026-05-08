/// <reference types="node" />

import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import mf from "./module-federation.config";
import tailwind from "@tailwindcss/vite";

import path from "path";
import { fileURLToPath } from "url";
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const plugins: PluginOption[] = [
  react(),
  tailwind(),
  mf as unknown as PluginOption
];

export default defineConfig({
  plugins,
  server: {
    fs: {
      allow: [path.resolve(__dirname, "..")]
    }
  },
  resolve: {
    alias: {
      "@shared": path.resolve(__dirname, "../shared")
    }
  }
});

