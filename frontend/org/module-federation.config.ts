// frontend/org/module-federation.config.ts
import { defineConfig } from "@module-federation/vite";
import pkgJson from "../../package.json" assert { type: "json" };

export default defineConfig({
  // ← ここがホスト remotes のキーと一致している必要あり
  name: "org",

  // ← ここが import("org/routes") の routes と一致
  exposes: {
    "./routes": "./src/routes.tsx",
  },

  shared: {
    react: { singleton: true, requiredVersion: (pkgJson as any).dependencies["react"] },
    "react-dom": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["react-dom"],
    },
    "react-router-dom": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["react-router-dom"],
    },
    "@tanstack/react-query": {
      singleton: true,
      requiredVersion: (pkgJson as any).dependencies["@tanstack/react-query"],
    },
  },

  filename: "remoteEntry.js",
  manifest: true,
  dts: false,
});
