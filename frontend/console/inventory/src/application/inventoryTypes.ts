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

  /**
   * RGB。
   * backend response では number | null を正とする。
   * 表示時は rgbIntToHex 等で hex 化して color dot に反映する。
   */
  rgb?: number | null;

  /** 在庫数 */
  stock: number;

  /**
   * 表示順。
   * backend の productBlueprintPatch.modelRefs[].displayOrder から付与する。
   * InventoryDetailRowDTO には含めず、application 側で表示用に合成する。
   */
  displayOrder?: number;
};