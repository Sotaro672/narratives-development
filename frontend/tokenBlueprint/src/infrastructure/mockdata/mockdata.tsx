// frontend/tokenBlueprint/src/infrastructure/mockdata/mockdata.tsx
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";

/**
 * TOKEN_BLUEPRINTS
 * frontend/shell/src/shared/types/tokenBlueprint.ts を正として構造を統一。
 * - iconId は URL 文字列をそのまま格納（モックデータとしての簡易表現）
 * - burnAt は backend 非対応フィールドのため、frontend 表示用に追加保持
 */
export const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  {
    id: "token_blueprint_001",
    name: "SILK Premium Token",
    symbol: "SILK",
    brandId: "LUMINA Fashion",
    description:
      "プレミアムシルクブラウスの購入者限定トークン。限定コンテンツおよび特別割引を提供します。",
    iconId:
      "https://images.pexels.com/photos/8437005/pexels-photo-8437005.jpeg?auto=compress&cs=tinysrgb&w=300",
    contentFiles: [],
    assigneeId: "佐藤 美咲",
    createdAt: "2024-01-20T00:00:00Z",
    createdBy: "佐藤 美咲",
    updatedAt: "2024-01-20T00:00:00Z",
    updatedBy: "佐藤 美咲",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_002",
    name: "NEXUS Street Token",
    symbol: "NEXUS",
    brandId: "NEXUS Street",
    description:
      "ストリートカルチャーと連動したブランド限定トークン。イベント参加やNFT特典に利用可能です。",
    iconId:
      "https://images.pexels.com/photos/6214476/pexels-photo-6214476.jpeg?auto=compress&cs=tinysrgb&w=300",
    contentFiles: [],
    assigneeId: "高橋 健太",
    createdAt: "2024-01-18T00:00:00Z",
    createdBy: "佐藤 美咲",
    updatedAt: "2024-01-18T00:00:00Z",
    updatedBy: "佐藤 美咲",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_003",
    name: "LUMINA VIP Token",
    symbol: "LVIP",
    brandId: "LUMINA Fashion",
    description:
      "LUMINAファッションブランドのVIP会員専用トークン。限定試着会・特典アクセスに使用されます。",
    iconId:
      "https://images.pexels.com/photos/7679650/pexels-photo-7679650.jpeg?auto=compress&cs=tinysrgb&w=300",
    contentFiles: [],
    assigneeId: "山田 太郎",
    createdAt: "2024-01-15T00:00:00Z",
    createdBy: "高橋 健太",
    updatedAt: "2024-01-15T00:00:00Z",
    updatedBy: "高橋 健太",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_004",
    name: "NEXUS Community Token",
    symbol: "NXCOM",
    brandId: "NEXUS Street",
    description:
      "NEXUSブランドのファンコミュニティ向けトークン。限定投稿やイベント参加に利用されます。",
    iconId:
      "https://images.pexels.com/photos/7619637/pexels-photo-7619637.jpeg?auto=compress&cs=tinysrgb&w=300",
    contentFiles: [],
    assigneeId: "佐藤 美咲",
    createdAt: "2024-01-12T00:00:00Z",
    createdBy: "高橋 健太",
    updatedAt: "2024-01-12T00:00:00Z",
    updatedBy: "高橋 健太",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_005",
    name: "SILK Limited Edition",
    symbol: "SLKED",
    brandId: "LUMINA Fashion",
    description:
      "LUMINAが提供する期間限定トークン。限定商品購入やブランド特典のアンロックに使用されます。",
    iconId:
      "https://images.pexels.com/photos/8454341/pexels-photo-8454341.jpeg?auto=compress&cs=tinysrgb&w=300",
    contentFiles: [],
    assigneeId: "高橋 健太",
    createdAt: "2024-01-10T00:00:00Z",
    createdBy: "佐藤 美咲",
    updatedAt: "2024-01-10T00:00:00Z",
    updatedBy: "佐藤 美咲",
    deletedAt: null,
    deletedBy: null,
  },
];
