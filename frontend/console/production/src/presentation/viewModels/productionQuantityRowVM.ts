// frontend/console/production/src/presentation/viewModels/productionQuantityRowVM.ts

/**
 * ProductionQuantityRowVM
 * ------------------------------------------------------------
 * Presentation 層の「正」となる ViewModel。
 * - UI は常に `modelId` をキーに扱う。
 * - ProductionQuantityCard などの UI コンポーネントは本 ViewModel のみに依存する。
 */
export type ProductionQuantityRowVM = {
  /** UI の一意キー。backend も modelId を正として扱う */
  modelId: string;

  /** 型番 */
  modelNumber: string;

  /** サイズ */
  size: string;

  /** カラー名 */
  color: string;

  /** RGB（0xRRGGBB int or string/nullable を許容） */
  rgb?: number;

  /**
   * 表示順の唯一のソース（ProductBlueprintDetail.modelRefs.displayOrder）
   * - 未設定の場合は undefined
   */
  displayOrder?: number;

  /** 生産数 */
  quantity: number;
};
