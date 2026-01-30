// frontend/console/production/src/presentation/create/types.ts

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
// - ProductionQuantityCard（application/detail 側）が参照する modelId/displayOrder に寄せる
// ======================================================================
export type ProductionQuantityRow = {
  /**
   * ✅ 正キー
   * ProductBlueprint.detail.modelRefs の modelId と join するキー
   * backend でも modelId のみを利用する
   */
  modelId: string;

  /** 型番（例: “GM”） */
  modelNumber: string;

  size: string;

  /** 色名（例: “グリーン”） */
  color: string;

  /** RGB 値（0xRRGGBB int） */
  rgb?: number | string | null;

  /** 表示順（ProductBlueprint.detail.modelRefs.displayOrder を注入） */
  displayOrder?: number;

  quantity: number;
};
