// frontend/list/src/infrastructure/mockdata/mockdata.tsx

import type { List, ListStatus } from "../../../../shell/src/shared/types/list";
import type {
  Discount,
  DiscountItem,
} from "../../../../shell/src/shared/types/discount";
import type {
  Sale,
  SalePrice,
} from "../../../../shell/src/shared/types/sale";

/**
 * モックデータ用 ListingRow。
 * List エンティティを UI 用に簡略化した表示構造。
 */
export interface ListingRow {
  id: string;
  productName: string;
  brandName: string;
  tokenName: string;
  stock: number;
  assigneeName: string;
  status: ListStatus; // "listing" | "suspended" | "deleted"
}

/**
 * 出品中リストのモックデータ。
 * backend/internal/domain/list/entity.go に基づく List の簡易表現。
 */
export const LISTINGS: ListingRow[] = [
  {
    id: "list_001",
    productName: "シルクブラウス プレミアムライン",
    brandName: "LUMINA Fashion",
    tokenName: "LUMINA VIP Token",
    stock: 221,
    assigneeName: "山田 太郎",
    status: "listing",
  },
  {
    id: "list_002",
    productName: "デニムジャケット ヴィンテージ加工",
    brandName: "NEXUS Street",
    tokenName: "NEXUS Community Token",
    stock: 222,
    assigneeName: "佐藤 美咲",
    status: "listing",
  },
  {
    id: "list_003",
    productName: "シルクブラウス プレミアムライン",
    brandName: "LUMINA Fashion",
    tokenName: "LUMINA VIP Token",
    stock: 221,
    assigneeName: "山田 太郎",
    status: "suspended",
  },
];

/**
 * モック用 DiscountItem データ
 * backend/internal/domain/discount/entity.go に準拠。
 */
export const DISCOUNT_ITEMS: DiscountItem[] = [
  { modelNumber: "model_001", discount: 10 },
  { modelNumber: "model_002", discount: 15 },
  { modelNumber: "model_003", discount: 5 },
];

/**
 * モック用 Discount データ
 * frontend/shell/src/shared/types/discount.ts に準拠。
 */
export const DISCOUNTS: Discount[] = [
  {
    id: "discount_001",
    listId: "list_001",
    discounts: [DISCOUNT_ITEMS[0], DISCOUNT_ITEMS[1]],
    description: "春の新作キャンペーン割引（最大15%OFF）",
    discountedBy: "member_001",
    discountedAt: "2024-03-10T10:00:00Z",
    updatedAt: "2024-03-10T10:00:00Z",
    updatedBy: "member_001",
  },
  {
    id: "discount_002",
    listId: "list_002",
    discounts: [DISCOUNT_ITEMS[2]],
    description: "数量限定セール（5%OFF）",
    discountedBy: "member_002",
    discountedAt: "2024-03-12T09:30:00Z",
    updatedAt: "2024-03-12T09:30:00Z",
    updatedBy: "member_002",
  },
];

/**
 * モック用 SalePrice データ
 * frontend/shell/src/shared/types/sale.ts に準拠。
 */
export const SALE_PRICES: SalePrice[] = [
  { modelNumber: "model_001", price: 12000 },
  { modelNumber: "model_002", price: 15000 },
  { modelNumber: "model_003", price: 9800 },
];

/**
 * モック用 Sale データ
 * frontend/shell/src/shared/types/sale.ts に準拠。
 *
 * - listId は LISTINGS / DISCOUNTS と対応
 * - discountId は存在する Discount.id を参照（または null）
 */
export const SALES: Sale[] = [
  {
    id: "sale_001",
    listId: "list_001",
    discountId: "discount_001",
    prices: [SALE_PRICES[0], SALE_PRICES[1]],
  },
  {
    id: "sale_002",
    listId: "list_002",
    discountId: "discount_002",
    prices: [SALE_PRICES[2]],
  },
  {
    id: "sale_003",
    listId: "list_003",
    discountId: null,
    prices: [SALE_PRICES[0]],
  },
];

/**
 * UI 向けステータス変換（ListStatus → 表示ラベル）
 */
export function getListStatusLabel(status: ListStatus): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    case "deleted":
      return "削除済み";
    default:
      return "不明";
  }
}

/**
 * UI 向けに List 型から ListingRow へ変換する補助関数。
 */
export function toListingRow(list: List): ListingRow {
  return {
    id: list.id,
    productName: "（商品名未設定）",
    brandName: "（ブランド未設定）",
    tokenName: "（トークン未設定）",
    stock: list.prices?.length ?? 0,
    assigneeName: list.assigneeId,
    status: list.status,
  };
}
