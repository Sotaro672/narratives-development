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
      // eslint-disable-next-line no-console
      console.log("[inventory/useInventoryDetail] reset (missing ids)", {
        pbId,
        tbId,
        productBlueprintIdRaw: productBlueprintId,
        tokenBlueprintIdRaw: tokenBlueprintId,
      });

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

        // eslint-disable-next-line no-console
        console.log("[inventory/useInventoryDetail] load start", { pbId, tbId });

        // ✅ 並列取得（token patch は失敗しても画面を落とさない）
        const mergedPromise = queryInventoryDetailByProductAndToken(pbId, tbId);
        const tokenPatchPromise = fetchTokenBlueprintPatchDTO(tbId)
          .then((p) => p)
          .catch((e: any) => {
            if (cancelled) return null;

            const msg = String(e?.message ?? e);

            // eslint-disable-next-line no-console
            console.warn("[inventory/useInventoryDetail] fetchTokenBlueprintPatchDTO failed", {
              tbId,
              error: msg,
              raw: e,
            });

            setTokenPatchError(msg);
            return null;
          });

        const [merged, tbPatch] = await Promise.all([mergedPromise, tokenPatchPromise]);

        if (cancelled) return;

        // eslint-disable-next-line no-console
        console.log("[inventory/useInventoryDetail] queryInventoryDetailByProductAndToken result", {
          inventoryKey: (merged as any)?.inventoryKey,
          pbId: (merged as any)?.productBlueprintId,
          tbId: (merged as any)?.tokenBlueprintId,
          productName: (merged as any)?.productName,
          brandId: (merged as any)?.brandId,
          brandName: (merged as any)?.brandName,
          rowsCount: Array.isArray((merged as any)?.rows) ? (merged as any).rows.length : 0,
          hasTokenBlueprintPatchInMerged: Boolean((merged as any)?.tokenBlueprintPatch),
          mergedTokenPatchKeys: Object.keys((merged as any)?.tokenBlueprintPatch ?? {}),
          mergedTokenPatchIconUrl: String((merged as any)?.tokenBlueprintPatch?.iconUrl ?? ""),
          mergedTokenPatchIconId: String((merged as any)?.tokenBlueprintPatch?.iconId ?? ""),
        });

        // eslint-disable-next-line no-console
        console.log("[inventory/useInventoryDetail] fetchTokenBlueprintPatchDTO result", {
          tbId,
          hasTbPatch: Boolean(tbPatch),
          tbPatchKeys: Object.keys((tbPatch as any) ?? {}),
          iconUrl: String((tbPatch as any)?.iconUrl ?? ""),
          iconId: String((tbPatch as any)?.iconId ?? ""),
          // よくある揺れも “存在チェック” だけ出す
          icon_url: String((tbPatch as any)?.icon_url ?? ""),
          iconURL: String((tbPatch as any)?.iconURL ?? ""),
          tokenIconUrl: String((tbPatch as any)?.tokenIconUrl ?? ""),
          tokenIcon: (tbPatch as any)?.tokenIcon ? Object.keys((tbPatch as any).tokenIcon ?? {}) : null,
        });

        const nextRows = Array.isArray(merged.rows) ? merged.rows : [];
        setVm(merged);
        setRows(nextRows);

        // ✅ TokenBlueprintCard 用 patch を伝送
        setTokenBlueprintPatch(tbPatch);

        // eslint-disable-next-line no-console
        console.log("[inventory/useInventoryDetail] state set", {
          rowsCount: nextRows.length,
          tokenBlueprintPatchIconUrl: String((tbPatch as any)?.iconUrl ?? ""),
        });
      } catch (e: any) {
        if (cancelled) return;

        const msg = String(e?.message ?? e);

        // eslint-disable-next-line no-console
        console.warn("[inventory/useInventoryDetail] load failed", {
          pbId,
          tbId,
          error: msg,
          raw: e,
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

        // eslint-disable-next-line no-console
        console.log("[inventory/useInventoryDetail] load end", { pbId, tbId });
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

    const out = {
      id,
      tokenName: String(p?.tokenName ?? p?.name ?? "").trim(),
      TokenName: String(p?.TokenName ?? "").trim(),
      symbol: String(p?.symbol ?? "").trim(),
      brandId: String(p?.brandId ?? "").trim(),
      brandName: String(p?.brandName ?? "").trim(),
      description: String(p?.description ?? "").trim(),
      minted: typeof p?.minted === "boolean" ? p.minted : false,
    };

    // eslint-disable-next-line no-console
    console.log("[inventory/useInventoryDetail] initialTokenBlueprintForCard computed", {
      tbId,
      hasPatch: Boolean(tokenBlueprintPatch),
      patchKeys: Object.keys(p ?? {}),
      out,
    });

    return out;
  }, [tbId, tokenBlueprintPatch]);

  const initialIconUrl = React.useMemo(() => {
    const url = String((tokenBlueprintPatch as any)?.iconUrl ?? "").trim();

    // eslint-disable-next-line no-console
    console.log("[inventory/useInventoryDetail] initialIconUrl computed", {
      tbId,
      hasPatch: Boolean(tokenBlueprintPatch),
      iconUrl: url,
      iconId: String((tokenBlueprintPatch as any)?.iconId ?? ""),
      patchKeys: Object.keys((tokenBlueprintPatch as any) ?? {}),
    });

    return url;
  }, [tbId, tokenBlueprintPatch]);

  const tokenCardHook = useTokenBlueprintCard({
    initialTokenBlueprint: initialTokenBlueprintForCard as any,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  // eslint-disable-next-line no-console
  console.log("[inventory/useInventoryDetail] tokenCardHook snapshot", {
    tbId,
    cardIconUrl: String((tokenCardHook as any)?.vm?.iconUrl ?? ""),
    remoteIconUrl: String((tokenCardHook as any)?.vm?.remoteIconUrl ?? ""),
    minted: Boolean((tokenCardHook as any)?.vm?.minted),
    isEditMode: Boolean((tokenCardHook as any)?.vm?.isEditMode),
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
