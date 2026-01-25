// frontend/console/inventory/src/presentation/hook/useInventoryDetail.tsx

import * as React from "react";
import type { InventoryRow } from "../../application/inventoryTypes";
import {
  queryInventoryDetailByProductAndToken,
  type InventoryDetailViewModel,
} from "../../application/inventoryDetail/inventoryDetailService";
import {
  fetchTokenBlueprintPatchDTO,
  type TokenBlueprintPatchDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

// ✅ TokenBlueprintCard hook を Inventory 側から利用（vm/handlers をそのまま渡す）
import { useTokenBlueprintCard } from "../../../../tokenBlueprint/src/presentation/hook/useTokenBlueprintCard";

export type UseInventoryDetailResult = {
  vm: InventoryDetailViewModel | null;
  rows: InventoryRow[];

  // ✅ TokenBlueprintCard 用（optional）
  tokenBlueprintPatch: TokenBlueprintPatchDTO | null;
  tokenPatchLoading: boolean;
  tokenPatchError: string | null;

  // ✅ TokenBlueprintCard hook 一式（useTokenBlueprintCard の結果を伝送）
  tokenCard: {
    vm: any;
    handlers: any;
    selectedIconFile: File | null;
  } | null;

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

            const msg = String(e?.message ?? e);
            setTokenPatchError(msg);
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

        const msg = String(e?.message ?? e);

        setError(msg);
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
  }, [pbId, tbId, productBlueprintId, tokenBlueprintId]);

  // ============================================================
  // ✅ TokenBlueprintCard 用の hook を Inventory 側で生成して返す
  // ============================================================
  const initialTokenBlueprintForCard = React.useMemo(() => {
    const id = tbId;
    const p: any = tokenBlueprintPatch ?? {};

    return {
      id,
      tokenName: String(p?.tokenName ?? p?.name ?? "").trim(),
      TokenName: String(p?.TokenName ?? "").trim(),
      symbol: String(p?.symbol ?? "").trim(),
      brandId: String(p?.brandId ?? "").trim(),
      brandName: String(p?.brandName ?? "").trim(),
      description: String(p?.description ?? "").trim(),
      minted: typeof p?.minted === "boolean" ? p.minted : false,
    };
  }, [tbId, tokenBlueprintPatch]);

  const initialIconUrl = React.useMemo(() => {
    return String((tokenBlueprintPatch as any)?.iconUrl ?? "").trim();
  }, [tbId, tokenBlueprintPatch]);

  const tokenCardHook = useTokenBlueprintCard({
    initialTokenBlueprint: initialTokenBlueprintForCard as any,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  const tokenCard = tbId
    ? {
        vm: tokenCardHook.vm,
        handlers: tokenCardHook.handlers,
        selectedIconFile: tokenCardHook.selectedIconFile,
      }
    : null;

  return {
    vm,
    rows,
    tokenBlueprintPatch,
    tokenPatchLoading,
    tokenPatchError,
    tokenCard,
    loading,
    error,
  };
}
