// frontend/console/production/src/presentation/create/types.ts

import type { ItemType, Fit } from "../../../../productBlueprint/src/domain/entity/catalog";

// ✅ domain を正にする（modelId/quantity の最小表現）
import type { ModelQuantity } from "../../../../production/src/domain/entity/production";

// ======================================================================
// ProductBlueprintCard（UI向け ViewModel）
// ======================================================================
export type ProductBlueprintForCard = {
  id: string;
  productName: string;
  brand?: string;

  itemType?: ItemType;
  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

// ======================================================================
// ProductionQuantityRow（UI 専用）
// - ✅ 正キーは modelId（domain と一致）
// - ✅ quantity は domain と一致
// - UI 表示に必要なメタ情報のみを追加
// ======================================================================
export type ProductionQuantityRow = ModelQuantity & {
  /** 型番（例: “GM”） */
  modelNumber: string;

  size: string;

  /** 色名（例: “グリーン”） */
  color: string;

  /** RGB 値（0xRRGGBB int） */
  rgb?: number | string | null;

  /** 表示順（ProductBlueprint.detail.modelRefs.displayOrder を注入） */
  displayOrder?: number;
};
