// frontend/inquiries/module-federation.config.ts
import { federation } from "@module-federation/vite";
import pkg from "../../package.json" assert { type: "json" };

export default {
  name: "inquiries",
  exposes: {
    "./routes": "./src/routes.tsx",
  },
  shared: {
    react: { singleton: true, requiredVersion: pkg.dependencies["react"] },
    "react-dom": { singleton: true, requiredVersion: pkg.dependencies["react-dom"] },
    "react-router-dom": { singleton: true, requiredVersion: pkg.dependencies["react-router-dom"] },
    "@tanstack/react-query": { singleton: true, requiredVersion: pkg.dependencies["@tanstack/react-query"] },
    "@apollo/client": { singleton: true, requiredVersion: pkg.dependencies["@apollo/client"] },
  },
  filename: "remoteEntry.js",
  manifest: true,
  dts: false,
};
