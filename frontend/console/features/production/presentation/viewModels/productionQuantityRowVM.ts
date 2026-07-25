// frontend/console/production/src/presentation/viewModels/productionQuantityRowVM.ts

/**
 * ProductionQuantityRowVM
 * ------------------------------------------------------------
 * Presentation 層の「正」となる ViewModel。
 *
 * - UI は常に `modelId` をキーに扱う。
 * - ProductionQuantityCard などの UI コンポーネントは本 ViewModel のみに依存する。
 * - apparel / alcohol の model variation を共通で扱う。
 */
export type ProductionQuantityRowVM = {
  /** UI の一意キー。backend も modelId を正として扱う */
  modelId: string;

  /** model variation kind */
  kind?: "apparel" | "alcohol" | string;

  /** 型番 */
  modelNumber: string;

  /**
   * UI 表示用の共通バリエーション名
   *
   * apparel: "M / Black"
   * alcohol: "720ml"
   */
  variationLabel?: string;

  /** サイズ: apparel 用 */
  size: string;

  /** カラー名: apparel 用 */
  color: string;

  /** RGB: apparel 用 */
  rgb?: number;

  /** 容量値: alcohol 用 */
  volumeValue?: number;

  /** 容量単位: alcohol 用 */
  volumeUnit?: string;

  /**
   * 表示順の唯一のソース（ProductBlueprintDetail.modelRefs.displayOrder）
   * - 未設定の場合は undefined
   */
  displayOrder?: number;

  /** 生産数 */
  quantity: number;
};