// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDeletedDetail.tsx
import * as React from "react";
import { useParams, useNavigate } from "react-router-dom";

// ★ 通常の詳細データ取得は既存 hook を利用（編集ハンドラは不要）
import { useProductBlueprintDetail } from "./useProductBlueprintDetail";

// 型
import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import type { Fit, ItemType } from "../../domain/entity/catalog";
import type {
  SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

// ★ 復旧用サービス（application 層）を利用
import { restoreDeletedProductBlueprint } from "../../application/productBlueprintDeletedDetailService";

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
 * - データ取得は useProductBlueprintDetail に委譲
 * - 復旧 / 物理削除だけここで保持
 */
export function useProductBlueprintDeletedDetail(): UseProductBlueprintDeletedDetailResult {
  const base = useProductBlueprintDetail();
  const { blueprintId } = useParams<{ blueprintId: string }>();
  const navigate = useNavigate();

  // ---------------------------------------------------
  // 復旧ボタン: deletedAt / deletedBy / expiredAt を null にする
  //   - application 層の restoreDeletedProductBlueprint 経由で呼ぶ
  // ---------------------------------------------------
  const handleRestore = React.useCallback(async () => {
    if (!blueprintId) {
      console.error(
        "[useProductBlueprintDeletedDetail] restore called with empty blueprintId",
      );
      alert("商品設計ID が不明です");
      return;
    }

    console.log(
      "[useProductBlueprintDeletedDetail] restore click: blueprintId=",
      blueprintId,
    );

    try {
      await restoreDeletedProductBlueprint(blueprintId);
      alert("復旧しました");

      // 復旧後は通常の一覧へ戻る
      navigate("/productBlueprint");
    } catch (err) {
      console.error(
        "[useProductBlueprintDeletedDetail] restore failed: blueprintId=",
        blueprintId,
        err,
      );
      alert("復旧に失敗しました");
    }
  }, [blueprintId, navigate]);

  // ---------------------------------------------------
  // 物理削除（未実装）
  // ---------------------------------------------------
  const handlePurge = React.useCallback(() => {
    if (!blueprintId) {
      console.error(
        "[useProductBlueprintDeletedDetail] purge called with empty blueprintId",
      );
      alert("商品設計ID が不明です");
      return;
    }

    const ok = window.confirm(
      "この商品設計を完全に削除しますか？\nこの操作は取り消せません。",
    );
    if (!ok) return;

    // TODO: purge API を接続
    alert("物理削除処理はまだ実装されていません。");
  }, [blueprintId]);

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
