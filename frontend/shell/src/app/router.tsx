import { defineConfig } from "@module-federation/vite";

export default defineConfig({
  name: "shell",
  remotes: {
    // remotes のキー = リモートの name
    // URL は Vite のデフォルト出力に合わせて /assets/remoteEntry.js
    org: "http://localhost:4001/assets/remoteEntry.js",
    inquiries: "http://localhost:4002/assets/remoteEntry.js",
    listings: "http://localhost:4003/assets/remoteEntry.js",
    operations: "http://localhost:4004/assets/remoteEntry.js",
    preview: "http://localhost:4005/assets/remoteEntry.js",
    production: "http://localhost:4006/assets/remoteEntry.js",
    design: "http://localhost:4007/assets/remoteEntry.js",
    mint: "http://localhost:4008/assets/remoteEntry.js",
    orders: "http://localhost:4009/assets/remoteEntry.js",
    ads: "http://localhost:4010/assets/remoteEntry.js",
    accounts: "http://localhost:4011/assets/remoteEntry.js",
    transactions: "http://localhost:4012/assets/remoteEntry.js",
  },
  shared: {
    react: { singleton: true },
    "react-dom": { singleton: true },
    "react-router-dom": { singleton: true },
    "@tanstack/react-query": { singleton: true },
  },
  exposes: {},
  filename: "remoteEntry.js",
  manifest: true,
  dts: false,
});
