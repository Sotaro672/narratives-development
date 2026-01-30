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
// - create flow では modelVariationId を保持し、保存payloadに利用する
// ======================================================================
export type ProductionQuantityRow = {
  /** create 保存用（ProductionCreate payload の modelVariationId） */
  modelVariationId: string;

  /**
   * ✅ 表示・並び替え用
   * ProductBlueprint.detail.modelRefs の modelId と join するキー
   * 現状のデータでは modelVariationId === modelId なので同値で持つ
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
