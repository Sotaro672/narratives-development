//frontend\console\production\src\presentation\create\types.ts
import type { ItemType, Fit } from "../../../../productBlueprint/src/domain/entity/catalog";

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
// ======================================================================
export type ProductionQuantityRow = {
  modelVariationId: string;

  /** 型番（例: “GM”） */
  modelNumber: string;

  size: string;

  /** 色名（例: “グリーン”） */
  color: string;

  /** RGB 値（0xRRGGBB int） */
  rgb?: number | string | null;

  quantity: number;
};
