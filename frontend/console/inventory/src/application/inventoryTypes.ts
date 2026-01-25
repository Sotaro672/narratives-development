// frontend/console/inventory/src/application/inventoryTypes.ts

export type InventoryRow = {
  /** トークン表示（例: "Token A" / "TB-xxxx" / tokenBlueprint name など） */
  token?: string;
  /** 型番 (例: "LM-SB-S-WHT") */
  modelNumber: string;
  /** サイズ (例: "S" | "M" | "L") */
  size: string;
  /** カラー表示名 (例: "ホワイト") */
  color: string;
  /** RGB - int(0xRRGGBB) で来ることもあるので、表示時に hex 化して dot に反映する */
  rgb?: number | string | null;
  /** 在庫数 */
  stock: number;
};
