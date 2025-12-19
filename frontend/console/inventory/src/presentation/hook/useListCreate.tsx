// frontend/console/inventory/src/presentation/hook/useListCreate.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ Admin 用 hook（担当者候補の取得・選択）
import { useAdminCard as useAdminCardHook } from "../../../../admin/src/presentation/hook/useAdminCard";

// ✅ NEW: PriceCard の row 型（state は hook 側で保持）
import type { PriceRow } from "../../../../list/src/presentation/components/priceCard";

// ✅ HTTP は repository に寄せる
import {
  fetchListCreateDTO,
  type ListCreateDTO,
} from "../../infrastructure/http/inventoryRepositoryHTTP";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export type ListingDecision = "list" | "hold";

export type UseListCreateResult = {
  title: string;
  onBack: () => void;
  onCreate: () => void;

  // dto
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;

  // display strings (already trimmed)
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;

  // price
  priceRows: PriceRow[];
  onChangePrice: (index: number, price: number | null) => void;

  // assignee
  assigneeName: string;
  assigneeCandidates: Array<{ id: string; name: string }>;
  loadingMembers: boolean;
  handleSelectAssignee: (id: string) => void;

  // decision
  decision: ListingDecision;
  setDecision: React.Dispatch<React.SetStateAction<ListingDecision>>;
};

export function useListCreate(): UseListCreateResult {
  const navigate = useNavigate();

  // ✅ routes.tsx で定義した param を受け取る（inventoryId or pb/tb）
  const params = useParams<{
    inventoryId?: string;
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  const inventoryId = React.useMemo(() => s(params.inventoryId), [params.inventoryId]);
  const productBlueprintId = React.useMemo(
    () => s(params.productBlueprintId),
    [params.productBlueprintId],
  );
  const tokenBlueprintId = React.useMemo(
    () => s(params.tokenBlueprintId),
    [params.tokenBlueprintId],
  );

  // eslint-disable-next-line no-console
  console.log("[inventory/useListCreate] params resolved", {
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    raw: {
      inventoryId: params.inventoryId,
      productBlueprintId: params.productBlueprintId,
      tokenBlueprintId: params.tokenBlueprintId,
    },
  });

  // ✅ PageHeader（title）には pb/tb を出さない
  const title = React.useMemo(() => {
    return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
  }, [inventoryId]);

  // eslint-disable-next-line no-console
  console.log("[inventory/useListCreate] title computed", { title, inventoryId });

  // ✅ 戻るは inventoryDetail へ絶対遷移
  const onBack = React.useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] onBack", {
      inventoryId,
      productBlueprintId,
      tokenBlueprintId,
    });

    // ✅ inventoryId ルートで来ていても、詳細は pb/tb で戻る
    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, inventoryId, productBlueprintId, tokenBlueprintId]);

  // ✅ 作成ボタン（PageHeader）
  const onCreate = React.useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] onCreate (stub)", {
      inventoryId,
      productBlueprintId,
      tokenBlueprintId,
    });

    // TODO: 出品作成APIを呼ぶ（今は仮）
    alert("作成しました（仮）");

    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, inventoryId, productBlueprintId, tokenBlueprintId]);

  // ============================================================
  // ✅ listCreate 用 DTO を取得（pb/tb または inventoryId から）
  // ============================================================
  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  // ✅ 手順A: DTOで inventoryId を得たら、URLを /inventory/list/create/:inventoryId に正規化する
  // - 無限ループ防止のため 1 回だけ実行
  const redirectedRef = React.useRef(false);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      // inventoryId が無い場合は pb/tb が必須
      const canFetch =
        Boolean(inventoryId) ||
        (Boolean(productBlueprintId) && Boolean(tokenBlueprintId));

      // eslint-disable-next-line no-console
      console.log("[inventory/useListCreate] load start", {
        canFetch,
        inventoryId,
        productBlueprintId,
        tokenBlueprintId,
      });

      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const input = {
          inventoryId: inventoryId || undefined,
          productBlueprintId: productBlueprintId || undefined,
          tokenBlueprintId: tokenBlueprintId || undefined,
        };

        // eslint-disable-next-line no-console
        console.log("[inventory/useListCreate] fetchListCreateDTO input", input);

        const data = await fetchListCreateDTO(input);

        if (cancelled) return;

        // eslint-disable-next-line no-console
        console.log("[inventory/useListCreate] fetchListCreateDTO result", {
          hasData: Boolean(data),
          keys: Object.keys((data as any) ?? {}),
          inventoryId: (data as any)?.inventoryId,
          productBlueprintId: (data as any)?.productBlueprintId,
          tokenBlueprintId: (data as any)?.tokenBlueprintId,
          productBrandName: (data as any)?.productBrandName,
          productName: (data as any)?.productName,
          tokenBrandName: (data as any)?.tokenBrandName,
          tokenName: (data as any)?.tokenName,
          raw: data,
        });

        // ✅ 手順A: inventoryId パスで来ていない場合、DTOで得た inventoryId へ置き換え
        const gotInventoryId = s((data as any)?.inventoryId);
        if (!inventoryId && gotInventoryId && !redirectedRef.current) {
          redirectedRef.current = true;

          // eslint-disable-next-line no-console
          console.log("[inventory/useListCreate] redirect to inventoryId route", {
            from: {
              inventoryId,
              productBlueprintId,
              tokenBlueprintId,
            },
            to: {
              inventoryId: gotInventoryId,
            },
          });

          // ✅ URL を正規化（history を汚さない）
          navigate(`/inventory/list/create/${encodeURIComponent(gotInventoryId)}`, {
            replace: true,
          });
        }

        setDTO(data);
      } catch (e) {
        if (cancelled) return;

        const msg = String(e instanceof Error ? e.message : e);

        // eslint-disable-next-line no-console
        console.warn("[inventory/useListCreate] fetchListCreateDTO failed", {
          inventoryId,
          productBlueprintId,
          tokenBlueprintId,
          error: msg,
          raw: e,
        });

        setDTOError(msg);
      } finally {
        if (cancelled) return;

        setLoadingDTO(false);

        // eslint-disable-next-line no-console
        console.log("[inventory/useListCreate] load end", {
          inventoryId,
          productBlueprintId,
          tokenBlueprintId,
        });
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [navigate, inventoryId, productBlueprintId, tokenBlueprintId]);

  const productBrandName = React.useMemo(() => s(dto?.productBrandName), [dto]);
  const productName = React.useMemo(() => s(dto?.productName), [dto]);
  const tokenBrandName = React.useMemo(() => s(dto?.tokenBrandName), [dto]);
  const tokenName = React.useMemo(() => s(dto?.tokenName), [dto]);

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] display strings computed", {
      productBrandName,
      productName,
      tokenBrandName,
      tokenName,
      hasDTO: Boolean(dto),
      dtoKeys: Object.keys((dto as any) ?? {}),
    });
  }, [productBrandName, productName, tokenBrandName, tokenName, dto]);

  // ============================================================
  // ✅ 左カラム：PriceCard（価格入力）
  // ============================================================
  // NOTE:
  // - いまの ListCreateDTO にはサイズ/カラー/在庫の行情報が無いので、初期は空配列。
  // - 将来、list-create DTO に rows を追加 or 別APIで rows を取れるようになったらここで set してください。
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] priceRows changed", {
      count: priceRows.length,
      sample: priceRows.slice(0, 5),
    });
  }, [priceRows]);

  const onChangePrice = React.useCallback((index: number, price: number | null) => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] onChangePrice", { index, price });

    setPriceRows((prev) => {
      const next = [...prev];
      if (!next[index]) return prev;
      next[index] = { ...next[index], price };
      return next;
    });
  }, []);

  // ============================================================
  // ✅ 右カラム：担当者選択（ボタンのみ表示）
  // ============================================================
  const { assigneeName, assigneeCandidates, loadingMembers, onSelectAssignee } =
    useAdminCardHook();

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] admin candidates snapshot", {
      assigneeName,
      loadingMembers: Boolean(loadingMembers),
      candidatesCount: Array.isArray(assigneeCandidates) ? assigneeCandidates.length : 0,
      sample: Array.isArray(assigneeCandidates)
        ? assigneeCandidates.slice(0, 5)
        : [],
    });
  }, [assigneeName, assigneeCandidates, loadingMembers]);

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      // eslint-disable-next-line no-console
      console.log("[inventory/useListCreate] handleSelectAssignee", { id });

      onSelectAssignee(id);
    },
    [onSelectAssignee],
  );

  // ============================================================
  // ✅ 出品｜保留 選択
  // ============================================================
  const [decision, setDecision] = React.useState<ListingDecision>("list");

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] decision changed", { decision });
  }, [decision]);

  return {
    title,
    onBack,
    onCreate,

    dto,
    loadingDTO,
    dtoError,

    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    priceRows,
    onChangePrice,

    assigneeName,
    assigneeCandidates: (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}
