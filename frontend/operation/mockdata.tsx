// frontend/operation/mockdata.tsx

export type TokenOperation = {
  tokenName: string;
  symbol: string;
  brand: string;
  linkedProducts: number; // 連携商品種類数
  manager: string;
  planned: number; // 計画量
  requested: number; // 申請量
  issued: number; // 発行量
  distributionRate: string; // "100.0%" のような文字列
};

export const TOKEN_OPERATIONS: TokenOperation[] = [
  {
    tokenName: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "山田 太郎",
    planned: 0,
    requested: 0,
    issued: 10,
    distributionRate: "100.0%",
  },
  {
    tokenName: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 10,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 0,
    requested: 0,
    issued: 12,
    distributionRate: "100.0%",
  },
  {
    tokenName: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "高橋 健太",
    planned: 0,
    requested: 5,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    linkedProducts: 0,
    manager: "高橋 健太",
    planned: 0,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
];
