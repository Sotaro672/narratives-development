// frontend/shell/module-federation.config.ts
// 型は import しない（@module-federation/vite は ModuleFederationOptions を export していないため）
import { federation } from "@module-federation/vite";

// ここで "shared" の requiredVersion: false のような boolean は NG です
// 必要なら文字列にする（"^18.0.0" など）or 配列にするのが安全
const mfOptions = {
  name: "shell",
  remotes: {
    member: "http://localhost:4001/assets/remoteEntry.js",
    brand: "http://localhost:4002/assets/remoteEntry.js",
    permission: "http://localhost:4003/assets/remoteEntry.js",
    // ... 略 ...
    company: "http://localhost:4019/assets/remoteEntry.js",
  },

  // ① シンプルに配列で共有（型的に一番安全）
  shared: [
    "react",
    "react-dom",
    "react-router-dom",
    "@tanstack/react-query",
    "@apollo/client",
  ],

  // ② もし詳細指定したい場合は下記のように（requiredVersion は string で）
  // shared: {
  //   react: { singleton: true, requiredVersion: "^18.0.0" },
  //   "react-dom": { singleton: true, requiredVersion: "^18.0.0" },
  //   "react-router-dom": { singleton: true, requiredVersion: "^6.0.0" },
  //   "@tanstack/react-query": { singleton: true },
  //   "@apollo/client": { singleton: true },
  // },
} satisfies Parameters<typeof federation>[0]; // ← federation() の第1引数の型に適合させる

export default mfOptions;
