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

function safePick(obj: any, keys: string[]): Record<string, any> {
  const out: Record<string, any> = {};
  for (const k of keys) out[k] = obj?.[k];
  return out;
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

  // ============================================================
  // 🔎 LOG: inputs
  // ============================================================
  React.useEffect(() => {
    console.log("[useInventoryDetail] inputs", {
      productBlueprintId,
      tokenBlueprintId,
      pbId,
      tbId,
    });
  }, [productBlueprintId, tokenBlueprintId, pbId, tbId]);

  React.useEffect(() => {
    // 入力不足なら reset
    if (!pbId || !tbId) {
      console.log("[useInventoryDetail] reset (missing ids)", { pbId, tbId });

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

        console.log("[useInventoryDetail] start fetch", { pbId, tbId });

        // ✅ 並列取得（token patch は失敗しても画面を落とさない）
        const mergedPromise = queryInventoryDetailByProductAndToken(pbId, tbId);

        const tokenPatchPromise = fetchTokenBlueprintPatchDTO(tbId)
          .then((p) => p)
          .catch((e: any) => {
            if (cancelled) return null;

            const msg = String(e?.message ?? e);
            console.warn("[useInventoryDetail] token patch fetch failed", {
              tbId,
              message: msg,
              error: e,
            });

            setTokenPatchError(msg);
            return null;
          });

        const [merged, tbPatch] = await Promise.all([mergedPromise, tokenPatchPromise]);

        if (cancelled) return;

        // ============================================================
        // 🔎 LOG: merged result + rows (what UI actually receives)
        // ============================================================
        const mergedAny: any = merged as any;
        console.log("[useInventoryDetail] merged vm (shape)", {
          keys: Object.keys(mergedAny ?? {}),
          inventoryId: mergedAny?.inventoryId,
          inventoryIdsCount: Array.isArray(mergedAny?.inventoryIds)
            ? mergedAny.inventoryIds.length
            : 0,
          productBlueprintId: mergedAny?.productBlueprintId,
          tokenBlueprintId: mergedAny?.tokenBlueprintId,
          modelId: mergedAny?.modelId,
          totalStock: mergedAny?.totalStock,
          rowsCount: Array.isArray(mergedAny?.rows) ? mergedAny.rows.length : 0,
        });

        const nextRows = Array.isArray(mergedAny?.rows) ? mergedAny.rows : [];
        console.log("[useInventoryDetail] rows sample", {
          count: nextRows.length,
          sample: nextRows.slice(0, 5),
          // row で本当に使っている（or 使いそうな）キーの確認
          pickedSample: nextRows.slice(0, 5).map((r: any) =>
            safePick(r, [
              "tokenBlueprintId",
              "token",
              "modelNumber",
              "size",
              "color",
              "rgb",
              "stock",
            ]),
          ),
        });

        // ============================================================
        // 🔎 LOG: token patch (what TokenBlueprintCard gets as source)
        // ============================================================
        const tbPatchAny: any = tbPatch as any;
        console.log("[useInventoryDetail] token blueprint patch", {
          exists: !!tbPatchAny,
          keys: tbPatchAny ? Object.keys(tbPatchAny) : [],
          picked: safePick(tbPatchAny, [
            "tokenName",
            "name",
            "TokenName",
            "symbol",
            "brandId",
            "brandName",
            "description",
            "minted",
            "metadataUri",
            "iconUrl",
          ]),
        });

        setVm(merged);
        setRows(nextRows);

        // ✅ TokenBlueprintCard 用 patch を伝送
        setTokenBlueprintPatch(tbPatch);
      } catch (e: any) {
        if (cancelled) return;

        const msg = String(e?.message ?? e);
        console.error("[useInventoryDetail] main fetch failed", {
          pbId,
          tbId,
          message: msg,
          error: e,
        });

        setError(msg);
        setVm(null);
        setRows([]);

        // main が落ちたら patch も reset
        setTokenBlueprintPatch(null);
      } finally {
        if (cancelled) return;

        setLoading(false);
        setTokenPatchLoading(false);

        console.log("[useInventoryDetail] done", {
          pbId,
          tbId,
        });
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

    const initial = {
      id,
      tokenName: String(p?.tokenName ?? p?.name ?? "").trim(),
      TokenName: String(p?.TokenName ?? "").trim(),
      symbol: String(p?.symbol ?? "").trim(),
      brandId: String(p?.brandId ?? "").trim(),
      brandName: String(p?.brandName ?? "").trim(),
      description: String(p?.description ?? "").trim(),
      minted: typeof p?.minted === "boolean" ? p.minted : false,
    };

    // ============================================================
    // 🔎 LOG: what is passed into useTokenBlueprintCard (initials)
    // ============================================================
    console.log("[useInventoryDetail] tokenCard initialTokenBlueprint", {
      tbId,
      initial,
      sourcePatchPicked: safePick(p, [
        "tokenName",
        "name",
        "TokenName",
        "symbol",
        "brandId",
        "brandName",
        "description",
        "minted",
        "iconUrl",
      ]),
    });

    return initial;
  }, [tbId, tokenBlueprintPatch]);

  const initialIconUrl = React.useMemo(() => {
    const url = String((tokenBlueprintPatch as any)?.iconUrl ?? "").trim();

    // 🔎 LOG: icon url used by TokenBlueprintCard
    console.log("[useInventoryDetail] tokenCard initialIconUrl", {
      tbId,
      iconUrl: url,
    });

    return url;
  }, [tbId, tokenBlueprintPatch]);

  const tokenCardHook = useTokenBlueprintCard({
    initialTokenBlueprint: initialTokenBlueprintForCard as any,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  // 🔎 LOG: tokenCard vm minimal surface (to shrink pickers later)
  React.useEffect(() => {
    if (!tbId) return;
    console.log("[useInventoryDetail] tokenCardHook vm (keys)", {
      tbId,
      keys: Object.keys((tokenCardHook as any)?.vm ?? {}),
      vmPicked: safePick((tokenCardHook as any)?.vm, [
        "id",
        "tokenName",
        "symbol",
        "brandId",
        "brandName",
        "description",
        "minted",
        "iconUrl",
        "editMode",
      ]),
    });
  }, [tbId, tokenCardHook.vm]);

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
