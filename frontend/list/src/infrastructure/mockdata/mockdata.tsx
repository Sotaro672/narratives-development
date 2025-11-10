// frontend/list/mockdata.tsx

import type { List, ListStatus } from "../../../../shell/src/shared/types/list";

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
