// frontend/console/list/src/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ✅ PriceCard hook（PriceRow 型もここから取り込む）
import { usePriceCard, type PriceRow } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ それ以外は service へ
import {
  resolveListDetailParams,
  loadListDetailDTO,
  deriveListDetail,
  computeListDetailPageTitle,
  useMainImageIndexGuard,
  useCancelledRef,
  type ListDetailRouteParams,
  type ListDetailDTO,
  s,
} from "../../application/listDetailService";

export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;

  // loading/error
  loading: boolean;
  error: string;

  // raw dto
  dto: ListDetailDTO | null;

  // listing (view)
  listingTitle: string;
  description: string;

  // decision/status (view)
  decision: "list" | "hold" | "" | string;

  // ✅ display strings (already trimmed)
  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  // images (view)
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  // price (PriceCard 用)
  priceRows: PriceRow[];
  priceCard: ReturnType<typeof usePriceCard>;

  // ✅ admin (view) : assigneeId + assigneeName を返す
  assigneeId: string;
  assigneeName: string;

  createdByName: string;
  createdAt: string;
};

export function useListDetail(): UseListDetailResult {
  const navigate = useNavigate();
  const params = useParams<ListDetailRouteParams>();

  const resolved = React.useMemo(() => resolveListDetailParams(params), [params]);
  const { listId, inventoryId } = resolved;

  // eslint-disable-next-line no-console
  console.log("[console/list/useListDetail] params resolved", {
    listId,
    inventoryId,
    raw: resolved.raw,
  });

  const onBack = React.useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[console/list/useListDetail] onBack");
    navigate(-1);
  }, [navigate]);

  // -----------------------------
  // Load DTO
  // -----------------------------
  const [dto, setDTO] = React.useState<ListDetailDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  const cancelledRef = useCancelledRef();

  React.useEffect(() => {
    const run = async () => {
      const id = s(listId);
      if (!id) {
        setDTO(null);
        setError("listId がありません（ルートパラメータを確認してください）。");
        return;
      }

      setLoading(true);
      setError("");

      try {
        // eslint-disable-next-line no-console
        console.log("[console/list/useListDetail] load start", { listId: id, inventoryId });

        const data = await loadListDetailDTO({ listId: id, inventoryIdHint: inventoryId });
        if (cancelledRef.current) return;

        // eslint-disable-next-line no-console
        console.log("[console/list/useListDetail] load success", {
          keys: Object.keys((data as any) ?? {}),
          id: (data as any)?.id,
          inventoryId: (data as any)?.inventoryId,
          assigneeId: (data as any)?.assigneeId,
          assigneeName: (data as any)?.assigneeName,
          productBrandId: (data as any)?.productBrandId,
          productBrandName: (data as any)?.productBrandName,
          productName: (data as any)?.productName,
          tokenBrandId: (data as any)?.tokenBrandId,
          tokenBrandName: (data as any)?.tokenBrandName,
          tokenName: (data as any)?.tokenName,
        });

        setDTO(data);
      } catch (e) {
        if (cancelledRef.current) return;
        const msg = String(e instanceof Error ? e.message : e);

        // eslint-disable-next-line no-console
        console.warn("[console/list/useListDetail] load failed", { msg, raw: e });
        setError(msg);
        setDTO(null);
      } finally {
        if (cancelledRef.current) return;
        setLoading(false);

        // eslint-disable-next-line no-console
        console.log("[console/list/useListDetail] load end", { listId });
      }
    };

    void run();
  }, [listId, inventoryId, cancelledRef]);

  // -----------------------------
  // Derived view fields (service)
  // -----------------------------
  const derived = React.useMemo(() => deriveListDetail<PriceRow>(dto), [dto]);

  const {
    listingTitle,
    description,
    decision,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls,
    priceRows,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,
  } = derived;

  // images
  const [mainImageIndex, setMainImageIndex] = React.useState(0);
  useMainImageIndexGuard({ imageUrls, mainImageIndex, setMainImageIndex });

  // ✅ PriceCard hook（view）
  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows,
    mode: "view",
    currencySymbol: "¥",
    showTotal: true,
    onChangePrice: undefined,
  });

  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log("[console/list/useListDetail] snapshot", {
      loading,
      hasDTO: Boolean(dto),
      decision,
      listingTitleLen: listingTitle.length,
      descriptionLen: description.length,
      imageUrlsCount: imageUrls.length,
      priceRowsCount: priceRows.length,

      productBrandId,
      productBrandName,
      productName,

      tokenBrandId,
      tokenBrandName,
      tokenName,

      assigneeId,
      assigneeName,
    });
  }, [
    loading,
    dto,
    decision,
    listingTitle,
    description,
    imageUrls.length,
    priceRows.length,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    assigneeId,
    assigneeName,
  ]);

  const pageTitle = React.useMemo(
    () => computeListDetailPageTitle({ listId, listingTitle }),
    [listId, listingTitle],
  );

  return {
    pageTitle,
    onBack,

    loading,
    error,

    dto,

    listingTitle,
    description,

    decision,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls,
    mainImageIndex,
    setMainImageIndex,

    priceRows,
    priceCard,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,
  };
}
