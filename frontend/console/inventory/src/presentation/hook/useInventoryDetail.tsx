// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../components/inventoryCard";
import {
  queryInventoryDetailByProductAndToken,
  type InventoryDetailViewModel,
} from "../../application/inventoryDetailService";
import {
  fetchTokenBlueprintPatchDTO,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];

  // ✅ TokenBlueprintCard 用（optional）
  tokenBlueprintPatch: TokenBlueprintPatchDTO | null;
  tokenPatchLoading: boolean;
  tokenPatchError: string | null;

  loading: boolean;
  error: string | null;
};

function asString(v: any): string {
  return String(v ?? "").trim();
}

export function useInventoryDetail(
  productBlueprintId: string | undefined,
  tokenBlueprintId: string | undefined,
): UseInventoryDetailResult {
  const [vm, setVm] = React.useState<InventoryDetailViewModel | null>(null);
  const [rows, setRows] = React.useState<InventoryRow[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  const [tokenBlueprintPatch, setTokenBlueprintPatch] =
    React.useState<TokenBlueprintPatchDTO | null>(null);
  const [tokenPatchLoading, setTokenPatchLoading] = React.useState(false);
  const [tokenPatchError, setTokenPatchError] = React.useState<string | null>(null);

  // ✅ 入力正規化は最小限
  const pbId = React.useMemo(() => asString(productBlueprintId), [productBlueprintId]);
  const tbId = React.useMemo(() => asString(tokenBlueprintId), [tokenBlueprintId]);

  React.useEffect(() => {
    // 入力不足なら reset
    if (!pbId || !tbId) {
      setVm(null);
      setRows([]);
      setError(null);
      setLoading(false);

      setTokenBlueprintPatch(null);
      setTokenPatchLoading(false);
      setTokenPatchError(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        setError(null);

        setTokenPatchLoading(true);
        setTokenPatchError(null);

        // ✅ 並列取得（token patch は失敗しても画面を落とさない）
        const mergedPromise = queryInventoryDetailByProductAndToken(pbId, tbId);
        const tokenPatchPromise = fetchTokenBlueprintPatchDTO(tbId)
          .then((p) => p)
          .catch((e: any) => {
            if (cancelled) return null;
            setTokenPatchError(String(e?.message ?? e));
            return null;
          });

        const [merged, tbPatch] = await Promise.all([mergedPromise, tokenPatchPromise]);

        if (cancelled) return;

        const nextRows = Array.isArray(merged.rows) ? merged.rows : [];
        setVm(merged);
        setRows(nextRows);

        // ✅ TokenBlueprintCard 用 patch を伝送
        setTokenBlueprintPatch(tbPatch);
      } catch (e: any) {
        if (cancelled) return;

        setError(String(e?.message ?? e));
        setVm(null);
        setRows([]);

        // main が落ちたら patch も reset
        setTokenBlueprintPatch(null);
      } finally {
        if (cancelled) return;
        setLoading(false);
        setTokenPatchLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [pbId, tbId]);

  return {
    vm,
    rows,
    tokenBlueprintPatch,
    tokenPatchLoading,
    tokenPatchError,
    loading,
    error,
  };
}
