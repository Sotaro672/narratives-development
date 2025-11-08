export type TokenBlueprint = {
  tokenBlueprintId: string; // 主キー
  name: string;
  symbol: string;
  brand: string;
  assignee: string; // 担当者
  createdBy: string; // 作成者
  createdAt: string; // YYYY/M/D
  iconUrl: string;   // トークンアイコン画像URL
  burnAt: string;    // 焼却予定日 YYYY-MM-DD
  description: string; // トークン説明文
};

export const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  {
    tokenBlueprintId: "token_blueprint_001",
    name: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    assignee: "佐藤 美咲",
    createdBy: "佐藤 美咲",
    createdAt: "2024/1/20",
    iconUrl:
      "https://images.pexels.com/photos/8437005/pexels-photo-8437005.jpeg?auto=compress&cs=tinysrgb&w=300",
    burnAt: "2025-12-31",
    description:
      "プレミアムシルクブラウスの購入者限定トークン。限定コンテンツおよび特別割引を提供します。",
  },
  {
    tokenBlueprintId: "token_blueprint_002",
    name: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    assignee: "高橋 健太",
    createdBy: "佐藤 美咲",
    createdAt: "2024/1/18",
    iconUrl:
      "https://images.pexels.com/photos/6214476/pexels-photo-6214476.jpeg?auto=compress&cs=tinysrgb&w=300",
    burnAt: "2025-11-30",
    description:
      "ストリートカルチャーと連動したブランド限定トークン。イベント参加やNFT特典に利用可能です。",
  },
  {
    tokenBlueprintId: "token_blueprint_003",
    name: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    assignee: "山田 太郎",
    createdBy: "高橋 健太",
    createdAt: "2024/1/15",
    iconUrl:
      "https://images.pexels.com/photos/7679650/pexels-photo-7679650.jpeg?auto=compress&cs=tinysrgb&w=300",
    burnAt: "2025-10-15",
    description:
      "LUMINAファッションブランドのVIP会員専用トークン。限定試着会・特典アクセスに使用されます。",
  },
  {
    tokenBlueprintId: "token_blueprint_004",
    name: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    assignee: "佐藤 美咲",
    createdBy: "高橋 健太",
    createdAt: "2024/1/12",
    iconUrl:
      "https://images.pexels.com/photos/7619637/pexels-photo-7619637.jpeg?auto=compress&cs=tinysrgb&w=300",
    burnAt: "2025-09-30",
    description:
      "NEXUSブランドのファンコミュニティ向けトークン。限定投稿やイベント参加に利用されます。",
  },
  {
    tokenBlueprintId: "token_blueprint_005",
    name: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    assignee: "高橋 健太",
    createdBy: "佐藤 美咲",
    createdAt: "2024/1/10",
    iconUrl:
      "https://images.pexels.com/photos/8454341/pexels-photo-8454341.jpeg?auto=compress&cs=tinysrgb&w=300",
    burnAt: "2025-08-31",
    description:
      "LUMINAが提供する期間限定トークン。限定商品購入やブランド特典のアンロックに使用されます。",
  },
];
