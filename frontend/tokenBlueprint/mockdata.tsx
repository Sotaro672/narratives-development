// frontend/tokenBlueprint/mockdata.tsx

export type TokenBlueprint = {
  name: string;
  symbol: string;
  brand: string;
  manager: string;
  createdAt: string; // YYYY/M/D
};

export const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  {
    name: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    createdAt: "2024/1/20",
  },
  {
    name: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    createdAt: "2024/1/18",
  },
  {
    name: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    manager: "山田 太郎",
    createdAt: "2024/1/15",
  },
  {
    name: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    manager: "佐藤 美咲",
    createdAt: "2024/1/12",
  },
  {
    name: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    manager: "高橋 健太",
    createdAt: "2024/1/10",
  },
];
