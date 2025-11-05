import { defineConfig, type PluginOption } from "vite";
import react from "@vitejs/plugin-react";
import mf from "./module-federation.config";
import tailwind from "@tailwindcss/vite";

const plugins: PluginOption[] = [
  react(),
  tailwind(),
  mf as unknown as PluginOption, // 型衝突を吸収
];

export default defineConfig({
  plugins: [react(), tailwind()],
})