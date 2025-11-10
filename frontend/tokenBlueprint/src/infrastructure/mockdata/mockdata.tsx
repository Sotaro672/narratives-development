// frontend/tokenBlueprint/src/infrastructure/mockdata/mockdata.tsx
// frontend/shell/src/shared/types/tokenBlueprint.ts / tokenIcon.ts を正としてモックデータを定義

import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";
import type { TokenIcon } from "../../../../shell/src/shared/types/tokenIcon";

/**
 * トークンアイコンのモックデータ
 * - TokenIcon 型に準拠
 * - 実際の画像 URL を url に設定
 * - fileName / size はダミー値
 */
export const TOKEN_ICONS: TokenIcon[] = [
  {
    id: "token_icon_silk_premium",
    url: "https://images.pexels.com/photos/8437005/pexels-photo-8437005.jpeg?auto=compress&cs=tinysrgb&w=300",
    fileName: "silk-premium-token.jpg",
    size: 120_000,
  },
  {
    id: "token_icon_nexus_street",
    url: "https://images.pexels.com/photos/6214476/pexels-photo-6214476.jpeg?auto=compress&cs=tinysrgb&w=300",
    fileName: "nexus-street-token.jpg",
    size: 115_000,
  },
  {
    id: "token_icon_lumina_vip",
    url: "https://images.pexels.com/photos/7679650/pexels-photo-7679650.jpeg?auto=compress&cs=tinysrgb&w=300",
    fileName: "lumina-vip-token.jpg",
    size: 110_000,
  },
  {
    id: "token_icon_nexus_community",
    url: "https://images.pexels.com/photos/7619637/pexels-photo-7619637.jpeg?auto=compress&cs=tinysrgb&w=300",
    fileName: "nexus-community-token.jpg",
    size: 105_000,
  },
  {
    id: "token_icon_silk_limited",
    url: "https://images.pexels.com/photos/8454341/pexels-photo-8454341.jpeg?auto=compress&cs=tinysrgb&w=300",
    fileName: "silk-limited-token.jpg",
    size: 100_000,
  },
];

/**
 * TOKEN_BLUEPRINTS
 * - iconId は TokenIcon.id を参照（URL 文字列ではなく ID として扱う）
 * - 他フィールドは shared/types/tokenBlueprint.ts に準拠
 */
export const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  {
    id: "token_blueprint_001",
    name: "SILK Premium Token",
    symbol: "SILK",
    brandId: "brand_001", // LUMINA Fashion
    description:
      "プレミアムシルクブラウスの購入者限定トークン。限定コンテンツおよび特別割引を提供します。",
    iconId: "token_icon_silk_premium",
    contentFiles: [],
    assigneeId: "member_001",
    createdAt: "2024-01-20T00:00:00Z",
    createdBy: "member_001",
    updatedAt: "2024-01-20T00:00:00Z",
    updatedBy: "member_001",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_002",
    name: "NEXUS Street Token",
    symbol: "NEXUS",
    brandId: "brand_002", // NEXUS Street
    description:
      "ストリートカルチャーと連動したブランド限定トークン。イベント参加やNFT特典に利用可能です。",
    iconId: "token_icon_nexus_street",
    contentFiles: [],
    assigneeId: "member_002",
    createdAt: "2024-01-18T00:00:00Z",
    createdBy: "member_001",
    updatedAt: "2024-01-18T00:00:00Z",
    updatedBy: "member_001",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_003",
    name: "LUMINA VIP Token",
    symbol: "LVIP",
    brandId: "brand_001", // LUMINA Fashion
    description:
      "LUMINAブランドのVIP会員専用トークン。限定試着会・特典アクセスに使用されます。",
    iconId: "token_icon_lumina_vip",
    contentFiles: [],
    assigneeId: "member_003",
    createdAt: "2024-01-15T00:00:00Z",
    createdBy: "member_002",
    updatedAt: "2024-01-15T00:00:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_004",
    name: "NEXUS Community Token",
    symbol: "NXCOM",
    brandId: "brand_002", // NEXUS Street
    description:
      "NEXUSブランドのファンコミュニティ向けトークン。限定投稿やイベント参加に利用されます。",
    iconId: "token_icon_nexus_community",
    contentFiles: [],
    assigneeId: "member_001",
    createdAt: "2024-01-12T00:00:00Z",
    createdBy: "member_002",
    updatedAt: "2024-01-12T00:00:00Z",
    updatedBy: "member_002",
    deletedAt: null,
    deletedBy: null,
  },
  {
    id: "token_blueprint_005",
    name: "SILK Limited Edition",
    symbol: "SLKED",
    brandId: "brand_001", // LUMINA Fashion
    description:
      "LUMINAが提供する期間限定トークン。限定商品購入やブランド特典のアンロックに使用されます。",
    iconId: "token_icon_silk_limited",
    contentFiles: [],
    assigneeId: "member_002",
    createdAt: "2024-01-10T00:00:00Z",
    createdBy: "member_001",
    updatedAt: "2024-01-10T00:00:00Z",
    updatedBy: "member_001",
    deletedAt: null,
    deletedBy: null,
  },
];
