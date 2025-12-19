// frontend/console/inventory/src/presentation/hook/useListCreate.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ Admin 用 hook（担当者候補の取得・選択）
import { useAdminCard as useAdminCardHook } from "../../../../admin/src/presentation/hook/useAdminCard";

// ✅ PriceCard hook（PriceRow 型もここから取り込む）
import { usePriceCard, type PriceRow } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ application service に移譲
import {
  resolveListCreateParams,
  computeListCreateTitle,
  canFetchListCreate,
  loadListCreateDTOFromParams,
  getInventoryIdFromDTO,
  shouldRedirectToInventoryIdRoute,
  buildInventoryListCreatePath,
  buildBackPath,
  buildAfterCreatePath,
  extractDisplayStrings,
  mapDTOToPriceRows,
  type ListCreateRouteParams,
  type ResolvedListCreateParams,
} from "../../application/listCreateService";

import type { ListCreateDTO } from "../../infrastructure/http/inventoryRepositoryHTTP";

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

  // price (PriceCard 用)
  priceRows: PriceRow[];
  onChangePrice: (index: number, price: number | null) => void;

  // ✅ PriceCard hook の結果
  priceCard: ReturnType<typeof usePriceCard>;

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

  // ✅ routes.tsx で定義した param を受け取る
  const params = useParams<ListCreateRouteParams>();

  // ✅ params の trim/正規化は service へ
  const resolvedParams: ResolvedListCreateParams = React.useMemo(
    () => resolveListCreateParams(params),
    [params],
  );

  const { inventoryId, productBlueprintId, tokenBlueprintId } = resolvedParams;

  // eslint-disable-next-line no-console
  console.log("[inventory/useListCreate] params resolved", {
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    raw: resolvedParams.raw,
  });

  // ✅ title 計算は service へ
  const title = React.useMemo(() => computeListCreateTitle(inventoryId), [inventoryId]);

  // eslint-disable-next-line no-console
  console.log("[inventory/useListCreate] title computed", { title, inventoryId });

  // ============================================================
  // ✅ 出品｜保留
  // ============================================================
  const [decision, setDecision] = React.useState<ListingDecision>("list");

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] decision changed", { decision });
  }, [decision]);

  // ============================================================
  // ✅ PriceRows（DTOから初期化し、以後はユーザー入力を保持）
  // ============================================================
  const [priceRows, setPriceRows] = React.useState<PriceRow[]>([]);
  const initializedPriceRowsRef = React.useRef(false);

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

  // ✅ PriceCard hook
  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows,
    mode: "edit",
    currencySymbol: "¥",
    showTotal: true,
    onChangePrice: (index, price, row) => {
      // eslint-disable-next-line no-console
      console.log("[inventory/useListCreate] priceCard.onChangePrice", {
        index,
        price,
        row,
      });
      onChangePrice(index, price);
    },
  });

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] priceCard snapshot", {
      isEdit: priceCard.isEdit,
      mode: priceCard.mode,
      totalStock: priceCard.totalStock,
      totalPrice: priceCard.totalPrice,
      rowsCount: priceCard.rowsVM.length,
      sample: priceCard.rowsVM.slice(0, 3).map((r) => ({
        key: r.key,
        size: r.size,
        color: r.color,
        stock: r.stock,
        bgColor: r.bgColor,
        priceInputValue: r.priceInputValue,
        priceDisplayText: r.priceDisplayText,
      })),
    });
  }, [priceCard]);

  // ============================================================
  // ✅ 戻る / 作成（path 組み立ては service へ）
  // ============================================================
  const onBack = React.useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] onBack", {
      inventoryId,
      productBlueprintId,
      tokenBlueprintId,
    });

    navigate(buildBackPath(resolvedParams));
  }, [navigate, inventoryId, productBlueprintId, tokenBlueprintId, resolvedParams]);

  const onCreate = React.useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[inventory/useListCreate] onCreate (stub)", {
      inventoryId,
      productBlueprintId,
      tokenBlueprintId,
      decision,
      priceRowsCount: priceRows.length,
      priceRowsSample: priceRows.slice(0, 5),
    });

    alert("作成しました（仮）");
    navigate(buildAfterCreatePath(resolvedParams));
  }, [
    navigate,
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    decision,
    priceRows,
    resolvedParams,
  ]);

  // eslint-disable-next-line no-console
  React.useEffect(() => {
    console.log("[inventory/useListCreate] create snapshot", {
      decision,
      priceRowsCount: priceRows.length,
    });
  }, [decision, priceRows]);

  // ============================================================
  // ✅ listCreate DTO 取得（service へ移譲）
  // ============================================================
  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  const redirectedRef = React.useRef(false);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      const canFetch = canFetchListCreate(resolvedParams);

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
        const data = await loadListCreateDTOFromParams(resolvedParams);
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
          priceRowsCount: Array.isArray((data as any)?.priceRows)
            ? (data as any).priceRows.length
            : 0,
          raw: data,
        });

        // ✅ inventoryId ルートへ正規化（手順A）
        const gotInventoryId = getInventoryIdFromDTO(data);
        if (
          shouldRedirectToInventoryIdRoute({
            currentInventoryId: inventoryId,
            gotInventoryId,
            alreadyRedirected: redirectedRef.current,
          })
        ) {
          redirectedRef.current = true;

          // eslint-disable-next-line no-console
          console.log("[inventory/useListCreate] redirect to inventoryId route", {
            from: { inventoryId, productBlueprintId, tokenBlueprintId },
            to: { inventoryId: gotInventoryId },
          });

          navigate(buildInventoryListCreatePath(gotInventoryId), { replace: true });
        }

        setDTO(data);

        // ✅ priceRows 初期化（DTOの modelResolver 結果を PriceCard に渡す）
        if (!initializedPriceRowsRef.current) {
          const nextRows = mapDTOToPriceRows(data);
          setPriceRows(nextRows);
          initializedPriceRowsRef.current = true;

          // eslint-disable-next-line no-console
          console.log("[inventory/useListCreate] init priceRows from dto", {
            count: nextRows.length,
            sample: nextRows.slice(0, 5),
          });
        }
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
  }, [navigate, inventoryId, productBlueprintId, tokenBlueprintId, resolvedParams]);

  // ✅ 表示文字列は service へ
  const { productBrandName, productName, tokenBrandName, tokenName } = React.useMemo(
    () => extractDisplayStrings(dto),
    [dto],
  );

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
  // ✅ 担当者選択
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
    priceCard,

    assigneeName,
    assigneeCandidates: (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}
