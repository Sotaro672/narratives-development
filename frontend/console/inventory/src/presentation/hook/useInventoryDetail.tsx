// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../components/inventoryCard";
import {
  queryInventoryDetailByProductAndToken,
  type InventoryDetailViewModel,
} from "../../application/inventoryDetailService";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];
  loading: boolean;
  error: string | null;
};

function asString(v: any): string {
  return String(v ?? "").trim();
}

// modelId を渡せない（現状 rows には modelNumber/size/color/rgb しか無い）ため、
// “modelId関連”として rows の中身（modelNumber/size/color/rgb）を観測できるログを用意する。
function summarizeModelFields(rows: InventoryRow[], limit = 20) {
  const safe = Array.isArray(rows) ? rows : [];
  const sample = safe.slice(0, limit).map((r, i) => ({
    i,
    modelNumber: asString((r as any)?.modelNumber),
    size: asString((r as any)?.size),
    color: asString((r as any)?.color),
    rgb: (r as any)?.rgb ?? null,
    rgbType: typeof ((r as any)?.rgb ?? null),
    stock: Number((r as any)?.stock ?? 0),
  }));

  const missing = {
    size: sample.filter((x) => !x.size || x.size === "-").length,
    color: sample.filter((x) => !x.color || x.color === "-").length,
    rgb: sample.filter((x) => x.rgb == null || x.rgb === "" || x.rgb === "-").length,
  };

  return { rowsCount: safe.length, sample, missing };
}

export function useInventoryDetail(
  productBlueprintId: string | undefined,
  tokenBlueprintId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // ✅ サービス層（queryInventoryDetailByProductAndToken）を正として、入力正規化は hook 側で最小限のみ
  const pbId = React.useMemo(() => asString(productBlueprintId), [productBlueprintId]);
  const tbId = React.useMemo(() => asString(tokenBlueprintId), [tokenBlueprintId]);

  React.useEffect(() => {
    // 入力不足なら何もしない（サービス層が正なので、ここでの補完ロジックは持たない）
    if (!pbId || !tbId) {
      setVm(null);
      setRows([]);
      setLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        // ✅ 方針A: pbId + tbId -> inventoryIds -> details -> merge(viewModel)
        // ✅ 取得・集計・揺れ吸収はサービス層を正として任せる
        const merged = await queryInventoryDetailByProductAndToken(pbId, tbId);

        if (cancelled) return;

        const nextRows = Array.isArray(merged.rows) ? merged.rows : [];

        // ✅ modelId関連ログ（modelNumber/size/color/rgb を観測）
        console.log("[inventory/useInventoryDetail] model fields summary", {
          inventoryKey: merged.inventoryKey,
          pbId: merged.productBlueprintId,
          tbId: merged.tokenBlueprintId,
          inventoryIdsCount: merged.inventoryIds?.length ?? 0,
          totalStock: merged.totalStock,
          updatedAt: merged.updatedAt,
          ...summarizeModelFields(nextRows, 20),
        });

        // さらに、rows の先頭数件を詳細に出したい場合（必要ならコメント解除）
        // nextRows.slice(0, 10).forEach((r, idx) => {
        //   console.log("[inventory/useInventoryDetail] model row", {
        //     idx,
        //     modelNumber: asString((r as any)?.modelNumber),
        //     size: asString((r as any)?.size),
        //     color: asString((r as any)?.color),
        //     rgb: (r as any)?.rgb ?? null,
        //     rgbType: typeof ((r as any)?.rgb ?? null),
        //     stock: Number((r as any)?.stock ?? 0),
        //   });
        // });

        setVm(merged);
        setRows(nextRows);
      } catch (e: any) {
        if (cancelled) return;

        const msg = String(e?.message ?? e);

        // ✅ modelId関連ログ（失敗時の入力を観測）
        console.error("[inventory/useInventoryDetail] fetch error (model fields)", {
          productBlueprintId: asString(productBlueprintId),
          tokenBlueprintId: asString(tokenBlueprintId),
          normalized: { pbId, tbId },
          error: msg,
          raw: e,
        });

        setError(msg);
        setVm(null);
        setRows([]);
      } finally {
        if (cancelled) return;
        setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [pbId, tbId, productBlueprintId, tokenBlueprintId]);

  return { vm, rows, loading, error };
}
