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

  // ✅ PageHeader（title）には pb/tb を出さない
  const title = React.useMemo(() => {
    return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
  }, [inventoryId]);

  // ✅ 戻るは inventoryDetail へ絶対遷移
  const onBack = React.useCallback(() => {
    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, productBlueprintId, tokenBlueprintId]);

  // ✅ 作成ボタン（PageHeader）
  const onCreate = React.useCallback(() => {
    // TODO: 出品作成APIを呼ぶ（今は仮）
    alert("作成しました（仮）");

    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, productBlueprintId, tokenBlueprintId]);

  // ============================================================
  // ✅ listCreate 用 DTO を取得（pb/tb または inventoryId から）
  // ============================================================
  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      // inventoryId が無い場合は pb/tb が必須
      const canFetch =
        Boolean(inventoryId) ||
        (Boolean(productBlueprintId) && Boolean(tokenBlueprintId));
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await fetchListCreateDTO({
          inventoryId: inventoryId || undefined,
          productBlueprintId: productBlueprintId || undefined,
          tokenBlueprintId: tokenBlueprintId || undefined,
        });
        if (!cancelled) setDTO(data);
      } catch (e) {
        if (!cancelled) setDTOError(String(e instanceof Error ? e.message : e));
      } finally {
        if (!cancelled) setLoadingDTO(false);
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [inventoryId, productBlueprintId, tokenBlueprintId]);

  const productBrandName = React.useMemo(() => s(dto?.productBrandName), [dto]);
  const productName = React.useMemo(() => s(dto?.productName), [dto]);
  const tokenBrandName = React.useMemo(() => s(dto?.tokenBrandName), [dto]);
  const tokenName = React.useMemo(() => s(dto?.tokenName), [dto]);

  // ============================================================
  // ✅ 左カラム：PriceCard（価格入力）
  // ============================================================
  // NOTE:
  // - いまの ListCreateDTO にはサイズ/カラー/在庫の行情報が無いので、初期は空配列。
  // - 将来、list-create DTO に rows を追加 or 別APIで rows を取れるようになったらここで set してください。
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);

  const onChangePrice = React.useCallback((index: number, price: number | null) => {
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

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      onSelectAssignee(id);
    },
    [onSelectAssignee],
  );

  // ============================================================
  // ✅ 出品｜保留 選択
  // ============================================================
  const [decision, setDecision] = React.useState<ListingDecision>("list");

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
