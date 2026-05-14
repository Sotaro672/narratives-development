// frontend/console/production/src/presentation/create/types.ts

import type { ProductBlueprintCategorySnapshot } from "../../../../productBlueprint/src/domain/entity/productBlueprintCategory";

// domain を正にする（modelId/quantity の最小表現）
import type { ModelQuantity } from "../../../../production/src/domain/entity/production";

// ======================================================================
// ProductBlueprintCard（UI向け ViewModel）
// ======================================================================
export type ProductBlueprintForCard = {
  id: string;
  productName: string;

  /**
   * ProductBlueprintCard が表示に使用するブランド名。
   */
  brandName?: string;

  /**
   * ProductBlueprintCard が期待する商品カテゴリ snapshot。
   *
   * 表示名は ProductBlueprintCard 側で
   * nameJa / nameEn / code / id などから解決する。
   */
  productBlueprintCategory: ProductBlueprintCategorySnapshot | null;

  /**
   * apparel 用の表示項目。
   */
  fit?: string;
  materials?: string;
  weight?: number;
  washTags?: string[];

  productIdTag?: string;
};

// ======================================================================
// ProductionQuantityRow（UI 専用）
// - 正キーは modelId（domain と一致）
// - quantity は domain と一致
// - UI 表示に必要なメタ情報のみを追加
// - apparel / alcohol の両方を扱えるようにする
// ======================================================================
export type ProductionQuantityRow = ModelQuantity & {
  /** model variation kind */
  kind?: "apparel" | "alcohol" | string;

  /** 型番（例: “GM”, “DAIGINJO-720”） */
  modelNumber: string;

  /**
   * 共通表示用バリエーション名。
   *
   * apparel: "M / グリーン"
   * alcohol: "720ml"
   */
  variationLabel: string;

  /** サイズ: apparel 用 */
  size?: string;

  /** 色名: apparel 用 */
  color?: string;

  /** RGB 値（0xRRGGBB int） */
  rgb?: number | string | null;

  /** 容量値: alcohol 用 */
  volumeValue?: number;

  /** 容量単位: alcohol 用 */
  volumeUnit?: string;

  /** 表示順（ProductBlueprint.detail.modelRefs.displayOrder を注入） */
  displayOrder?: number;
};