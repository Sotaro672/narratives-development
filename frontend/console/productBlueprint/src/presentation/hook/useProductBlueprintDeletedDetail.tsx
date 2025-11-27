import * as React from "react";
import { useProductBlueprintDetail } from "./useProductBlueprintDetail";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import type { Fit, ItemType } from "../../domain/entity/catalog";
import type {
  SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

export interface UseProductBlueprintDeletedDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;
  itemType: ItemType | "";
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: ProductIDTagType | "";

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  colorRgbMap: Record<string, string>;

  assignee: string;
  creator: string;
  createdAt: string;

  getCode: (sizeLabel: string, color: string) => string;

  onBack: () => void;
  onRestore: () => void;
  onPurge: () => void;
}

/**
 * 削除済み商品設計 詳細画面用 hook
 * - 中身のデータ取得は useProductBlueprintDetail に委譲
 * - 復旧 / 物理削除ボタンのハンドラだけここで定義
 */
export function useProductBlueprintDeletedDetail(): UseProductBlueprintDeletedDetailResult {
  const base = useProductBlueprintDetail();

  // ----------------------------------------
  // 復旧 / 物理削除ボタン用ハンドラ
  // （現時点では API 未接続なので TODO としてアラートのみ）
  // ----------------------------------------
  const handleRestore = React.useCallback(() => {
    // TODO: 復旧 API と接続（restoreProductBlueprint など）
    alert("復旧処理はまだ実装されていません。");
  }, []);

  const handlePurge = React.useCallback(() => {
    // TODO: 物理削除 API と接続（purgeProductBlueprint など）
    const ok = window.confirm(
      "この商品設計を完全に削除しますか？\nこの操作は取り消せません。",
    );
    if (!ok) return;

    alert("物理削除処理はまだ実装されていません。");
  }, []);

  return {
    pageTitle: base.pageTitle,

    productName: base.productName,
    brand: base.brand,
    itemType: base.itemType,
    fit: base.fit,
    materials: base.materials,
    weight: base.weight,
    washTags: base.washTags,
    productIdTag: base.productIdTag as ProductIDTagType | "",

    colors: base.colors,
    colorInput: base.colorInput,
    sizes: base.sizes as SizeRow[],
    colorRgbMap: base.colorRgbMap,

    assignee: base.assignee,
    creator: base.creator,
    createdAt: base.createdAt,

    getCode: base.getCode,

    onBack: base.onBack,
    onRestore: handleRestore,
    onPurge: handlePurge,
  };
}
