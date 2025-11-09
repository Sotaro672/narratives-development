// frontend/productBlueprint/mockdata.tsx
import type {
  ProductBlueprint,
  ProductIDTagType,
  ItemType,
} from "../../../../shell/src/shared/types/productBlueprint";

/**
 * ProductBlueprint のモックデータ行。
 * UIリスト表示・テーブル用途に使用。
 */
export type ProductBlueprintRow = Pick<
  ProductBlueprint,
  "id" | "productName" | "brandId" | "assigneeId" | "createdAt" | "productIdTag"
> & {
  brandName: string;
  assigneeName: string;
};

/**
 * モックデータ一覧
 * backend/internal/domain/productBlueprint/entity.go と整合性を持つ形で表現。
 */
export const RAW_ROWS: ProductBlueprintRow[] = [
  {
    id: "pb_001",
    productName: "シルクブラウス プレミアムライン",
    brandId: "brand_001",
    brandName: "LUMINA Fashion",
    assigneeId: "member_001",
    assigneeName: "佐藤 美咲",
    productIdTag: { type: "qr" satisfies ProductIDTagType },
    createdAt: "2024-01-15T00:00:00Z",
  },
  {
    id: "pb_002",
    productName: "デニムジャケット ヴィンテージ加工",
    brandId: "brand_002",
    brandName: "NEXUS Street",
    assigneeId: "member_002",
    assigneeName: "高橋 健太",
    productIdTag: { type: "qr" satisfies ProductIDTagType },
    createdAt: "2024-01-10T00:00:00Z",
  },
];

/**
 * 製品設計のダミー詳細データ
 * 一覧→詳細の動作検証に使用可能。
 */
export const PRODUCT_BLUEPRINTS: ProductBlueprint[] = [
  {
    id: "pb_001",
    productName: "シルクブラウス プレミアムライン",
    brandId: "brand_001",
    itemType: "tops" satisfies ItemType,
    variations: [
      { id: "v001", name: "ホワイト S" },
      { id: "v002", name: "ホワイト M" },
    ],
    fit: "レギュラーフィット",
    material: "シルク100%",
    weight: 0.32,
    qualityAssurance: ["検品済み", "防シワ加工"],
    productIdTag: { type: "qr" },
    assigneeId: "member_001",
    createdBy: "member_001",
    createdAt: "2024-01-15T00:00:00Z",
    lastModifiedAt: "2024-01-15T00:00:00Z",
  },
  {
    id: "pb_002",
    productName: "デニムジャケット ヴィンテージ加工",
    brandId: "brand_002",
    itemType: "tops" satisfies ItemType,
    variations: [
      { id: "v101", name: "インディゴ M" },
      { id: "v102", name: "インディゴ L" },
    ],
    fit: "リラックスフィット",
    material: "コットン100%",
    weight: 1.05,
    qualityAssurance: ["色落ちテスト済み"],
    productIdTag: { type: "qr" },
    assigneeId: "member_002",
    createdBy: "member_002",
    createdAt: "2024-01-10T00:00:00Z",
    lastModifiedAt: "2024-01-10T00:00:00Z",
  },
];
