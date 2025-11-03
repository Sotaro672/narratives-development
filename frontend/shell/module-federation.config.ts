// ※ 型importはしない（ModuleFederationOptions は未エクスポートのため）
import pkg from "../../package.json";

const deps = (pkg as any).dependencies as Record<string, string>;

const mfOptions = {
  name: "shell",
  remotes: {
    member: "http://localhost:4001/assets/remoteEntry.js",
    brand: "http://localhost:4002/assets/remoteEntry.js",
    permission: "http://localhost:4003/assets/remoteEntry.js",
    inquiry: "http://localhost:4004/assets/remoteEntry.js",

    list: "http://localhost:4005/assets/remoteEntry.js",
    operation: "http://localhost:4006/assets/remoteEntry.js",
    preview: "http://localhost:4007/assets/remoteEntry.js",
    production: "http://localhost:4008/assets/remoteEntry.js",

    tokenBlueprint: "http://localhost:4009/assets/remoteEntry.js",
    mint: "http://localhost:4010/assets/remoteEntry.js",
    order: "http://localhost:4011/assets/remoteEntry.js",
    ads: "http://localhost:4012/assets/remoteEntry.js",
    account: "http://localhost:4013/assets/remoteEntry.js",
    transaction: "http://localhost:4014/assets/remoteEntry.js",
    inventory: "http://localhost:4015/assets/remoteEntry.js",
    message: "http://localhost:4016/assets/remoteEntry.js",
    announce: "http://localhost:4017/assets/remoteEntry.js",
    productBlueprint: "http://localhost:4018/assets/remoteEntry.js",
    company: "http://localhost:4019/assets/remoteEntry.js",
  },

  // shared をこうする（requiredVersion を外す or false に）
shared: {
  react: { singleton: true, requiredVersion: false },
  "react-dom": { singleton: true, requiredVersion: false },
  "react-router-dom": { singleton: true, requiredVersion: false },
  "@tanstack/react-query": { singleton: true, requiredVersion: false },
  "@apollo/client": { singleton: true, requiredVersion: false },
},

  exposes: {},
  filename: "remoteEntry.js",
  manifest: true,
  dts: false,
} as const;

export default mfOptions;
