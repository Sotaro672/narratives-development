// frontend/inventory/src/infrastructure/mockdata/mockdata.tsx

import type { Inventory } from "../../../../shell/src/shared/types/inventory";

/**
 * InventoryRow
 * 在庫一覧画面などで利用する簡易データ行。
 * backend/internal/domain/inventory/entity.go に基づくが、
 * 表示用途に特化して簡略化した構造。
 */
export interface InventoryRow {
  id: string;
  productName: string;
  brandName: string;
  tokenName: string | null;
  totalQuantity: number;
  status: "inspecting" | "inspected" | "listed" | "discarded" | "deleted";
}

/**
 * INVENTORIES
 * 在庫モックデータ。
 * 合計数量（totalQuantity）は models の合計値を表す。
 */
export const INVENTORIES: InventoryRow[] = [
  {
    id: "inv_001",
    productName: "シルクブラウス プレミアムライン",
    brandName: "LUMINA Fashion",
    tokenName: "LUMINA VIP Token",
    totalQuantity: 221,
    status: "listed",
  },
  {
    id: "inv_002",
    productName: "デニムジャケット ヴィンテージ加工",
    brandName: "NEXUS Street",
    tokenName: "NEXUS Community Token",
    totalQuantity: 222,
    status: "inspected",
  },
];

/**
 * INVENTORY_DETAILS
 * 各在庫の詳細データ（Inventory 構造準拠）。
 * shell/shared/types/inventory.ts に定義される Inventory を正とする。
 */
export const INVENTORY_DETAILS: Inventory[] = [
  {
    id: "inv_001",
    connectedToken: "LUMINA-VIP-TOKEN-001",
    models: [
      { modelNumber: "LB-001-S-WH", quantity: 100 },
      { modelNumber: "LB-001-M-WH", quantity: 80 },
      { modelNumber: "LB-001-L-WH", quantity: 41 },
    ],
    location: "Tokyo Warehouse A",
    status: "listed",
    createdBy: "member_001",
    createdAt: "2024-04-01T09:00:00Z",
    updatedBy: "member_001",
    updatedAt: "2024-04-10T10:00:00Z",
  },
  {
    id: "inv_002",
    connectedToken: "NEXUS-COMMUNITY-TOKEN-002",
    models: [
      { modelNumber: "NJ-101-S-BL", quantity: 70 },
      { modelNumber: "NJ-101-M-BL", quantity: 100 },
      { modelNumber: "NJ-101-L-BL", quantity: 52 },
    ],
    location: "Osaka Main Storage",
    status: "inspected",
    createdBy: "member_002",
    createdAt: "2024-05-05T10:00:00Z",
    updatedBy: "member_002",
    updatedAt: "2024-05-15T11:00:00Z",
  },
];

