// frontend/console/inventory/src/application/inventoryTypes.ts

export type InventoryModelKind = "apparel" | "alcohol" | string;

export type InventoryRow = {
  /** トークン表示（例: "Token A" / "TB-xxxx" / tokenBlueprint name など） */
  token?: string;

  /**
   * モデル種別。
   *
   * ProductBlueprintCategory.kind や model row の種類から付与する。
   * - apparel: size / color / rgb を表示
   * - alcohol: volumeValue / volumeUnit を表示
   */
  kind?: InventoryModelKind | null;

  /** 型番 (例: "LM-SB-S-WHT") */
  modelNumber: string;

  /** サイズ (例: "S" | "M" | "L") */
  size?: string | null;

  /** カラー表示名 (例: "ホワイト") */
  color?: string | null;

  /**
   * RGB。
   * backend response では number | null を正とする。
   * 表示時は rgbIntToHex 等で hex 化して color dot に反映する。
   */
  rgb?: number | null;

  /**
   * 容量。
   *
   * alcohol category の場合に使用する。
   * 例: 720
   */
  volumeValue?: number | null;

  /**
   * 容量単位。
   *
   * alcohol category の場合に使用する。
   * 例: "ml" / "L"
   */
  volumeUnit?: string | null;

  /** 在庫数 */
  stock: number;

  /**
   * 表示順。
   * backend の productBlueprintPatch.modelRefs[].displayOrder から付与する。
   * InventoryDetailRowDTO には含めず、application 側で表示用に合成する。
   */
  displayOrder?: number;
};