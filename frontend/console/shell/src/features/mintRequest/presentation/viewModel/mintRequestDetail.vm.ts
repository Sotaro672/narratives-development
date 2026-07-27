// frontend/console/shell/src/features/mintRequest/presentation/viewModel/mintRequestDetail.vm.ts

// ============================================================
// ViewModel Types for MintRequestDetail
// ============================================================

import type { BrandSummary } from "../../application/port/MintRequestRepository";

import type { ProductBlueprintCategorySnapshot } from "../../../productBlueprint/domain/productBlueprintCategory";

export type ProductBlueprintCardVM = {
  productName?: string;

  /**
   * ブランドの表示名。
   *
   * IDや別名は保持せず、
   * UI表示ではbrandNameのみを使用する。
   */
  brandName?: string;

  /**
   * 商品カテゴリsnapshot。
   *
   * ProductBlueprintCardは
   * productBlueprintCategoryを参照して
   * 商品カテゴリを表示する。
   */
  productBlueprintCategory?:
    | ProductBlueprintCategorySnapshot
    | null;
};

export type TokenBlueprintCardVM = {
  id: string;

  /**
   * トークン名。
   *
   * MintRequest内ではtokenNameを正とし、
   * nameは使用しない。
   */
  tokenName: string;

  symbol: string;

  /**
   * 共通TokenBlueprintCardへ渡す
   * ブランドID。
   */
  brandId: string;

  /**
   * UI表示用のブランド名。
   */
  brandName: string;

  description: string;
  iconUrl?: string;

  isEditMode: boolean;

  /**
   * ブランド候補。
   */
  brandOptions: BrandSummary[];
};

export type TokenBlueprintCardHandlersVM = {
  onPreview: () => void;
};