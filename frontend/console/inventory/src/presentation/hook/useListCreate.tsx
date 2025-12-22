// frontend/console/inventory/src/presentation/hook/useListCreate.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ★ Admin 用 hook（担当者候補の取得・選択）
import { useAdminCard as useAdminCardHook } from "../../../../admin/src/presentation/hook/useAdminCard";

// ✅ PriceCard hook（PriceRow 型もここから取り込む）
import {
  usePriceCard,
  type PriceRow,
} from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ application service に移譲（hook には state/handler のみ残す）
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
  initPriceRowsFromDTO,
  createListWithImages,
  dedupeFiles,
  type ListCreateRouteParams,
  type ResolvedListCreateParams,
  type ImageInputRef,
  type PriceRowEx,
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
  priceRows: PriceRowEx[];
  onChangePrice: (index: number, price: number | null) => void;

  // ✅ PriceCard hook の結果
  priceCard: ReturnType<typeof usePriceCard>;

  // listing local states
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;

  // images
  images: File[];
  imagePreviewUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;
  imageInputRef: ImageInputRef; // ✅ null を含む RefObject
  onSelectImages: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onDropImages: (e: React.DragEvent<HTMLDivElement>) => void;
  onDragOverImages: (e: React.DragEvent<HTMLDivElement>) => void;
  removeImageAt: (idx: number) => void;
  clearImages: () => void;

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

  const params = useParams<ListCreateRouteParams>();

  const resolvedParams: ResolvedListCreateParams = React.useMemo(
    () => resolveListCreateParams(params),
    [params],
  );

  const { inventoryId } = resolvedParams;

  const title = React.useMemo(() => computeListCreateTitle(inventoryId), [inventoryId]);

  // ============================================================
  // ✅ 出品｜保留
  // ============================================================
  const [decision, setDecision] = React.useState<ListingDecision>("list");

  // ============================================================
  // ✅ listing fields
  // ============================================================
  const [listingTitle, setListingTitle] = React.useState<string>("");
  const [description, setDescription] = React.useState<string>("");

  // ============================================================
  // ✅ images
  // ============================================================
  const [images, setImages] = React.useState<File[]>([]);
  const [mainImageIndex, setMainImageIndex] = React.useState<number>(0);

  const imageInputRef = React.useRef<HTMLInputElement | null>(null);

  const onSelectImages = React.useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []).filter(Boolean) as File[];
    if (files.length === 0) return;

    setImages((prev) => dedupeFiles(prev, files));

    // 同じファイルを再選択できるように
    e.currentTarget.value = "";
  }, []);

  const onDropImages = React.useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();

    const files = Array.from(e.dataTransfer.files ?? [])
      .filter(Boolean)
      .filter((f) => String(f.type || "").startsWith("image/")) as File[];

    if (files.length === 0) return;

    setImages((prev) => dedupeFiles(prev, files));
  }, []);

  const onDragOverImages = React.useCallback((e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const removeImageAt = React.useCallback((idx: number) => {
    setImages((prev) => prev.filter((_, i) => i !== idx));

    setMainImageIndex((prevMain) => {
      if (idx === prevMain) return 0;
      if (idx < prevMain) return Math.max(0, prevMain - 1);
      return prevMain;
    });
  }, []);

  const clearImages = React.useCallback(() => {
    setImages([]);
    setMainImageIndex(0);
  }, []);

  // preview urls
  const [imagePreviewUrls, setImagePreviewUrls] = React.useState<string[]>([]);
  React.useEffect(() => {
    if (images.length === 0) {
      setImagePreviewUrls([]);
      return;
    }

    const urls = images.map((f) => URL.createObjectURL(f));
    setImagePreviewUrls(urls);

    return () => {
      urls.forEach((u) => {
        try {
          URL.revokeObjectURL(u);
        } catch {
          // noop
        }
      });
    };
  }, [images]);

  // main index guard
  React.useEffect(() => {
    if (images.length === 0) {
      if (mainImageIndex !== 0) setMainImageIndex(0);
      return;
    }
    if (mainImageIndex < 0 || mainImageIndex > images.length - 1) {
      setMainImageIndex(0);
    }
  }, [images.length, mainImageIndex]);

  // ============================================================
  // ✅ PriceRows
  // ============================================================
  const [priceRows, setPriceRows] = React.useState<PriceRowEx[]>([]);
  const initializedPriceRowsRef = React.useRef(false);

  const onChangePrice = React.useCallback((index: number, price: number | null) => {
    setPriceRows((prev) => {
      const next = [...prev];
      if (!next[index]) return prev;
      next[index] = { ...next[index], price };
      return next;
    });
  }, []);

  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows as unknown as PriceRow[],
    mode: "edit",
    currencySymbol: "¥",
    showTotal: true,
    onChangePrice: (index, price) => onChangePrice(index, price),
  });

  // ============================================================
  // ✅ 戻る
  // ============================================================
  const onBack = React.useCallback(() => {
    navigate(buildBackPath(resolvedParams));
  }, [navigate, resolvedParams]);

  // ============================================================
  // ✅ 担当者
  // ============================================================
  const { assigneeName, assigneeCandidates, loadingMembers, onSelectAssignee } =
    useAdminCardHook();

  const [assigneeId, setAssigneeId] = React.useState<string | undefined>(undefined);

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      setAssigneeId(id || undefined);
      onSelectAssignee(id);
    },
    [onSelectAssignee],
  );

  // ============================================================
  // ✅ 作成（listImage関連ログのみ残す）
  // ============================================================
  const onCreate = React.useCallback(() => {
    void (async () => {
      try {
        if (images.length > 0) {
          // eslint-disable-next-line no-console
          console.log("[inventory/listImage] create start", {
            imagesCount: images.length,
            mainImageIndex,
          });
        }

        await createListWithImages({
          params: resolvedParams,
          listingTitle,
          description,
          priceRows: priceRows as any,
          decision,
          assigneeId,
          images,
          mainImageIndex,
        });

        alert("作成しました");
        navigate(buildAfterCreatePath(resolvedParams));
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);

        if (images.length > 0) {
          // eslint-disable-next-line no-console
          console.log("[inventory/listImage] create failed", { msg });
        }

        alert(msg);
      }
    })();
  }, [
    decision,
    description,
    images,
    listingTitle,
    mainImageIndex,
    navigate,
    priceRows,
    resolvedParams,
    assigneeId,
  ]);

  // ============================================================
  // ✅ listCreate DTO 取得
  // ============================================================
  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  const redirectedRef = React.useRef(false);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      const canFetch = canFetchListCreate(resolvedParams);
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await loadListCreateDTOFromParams(resolvedParams);
        if (cancelled) return;

        const gotInventoryId = getInventoryIdFromDTO(data);
        if (
          shouldRedirectToInventoryIdRoute({
            currentInventoryId: inventoryId,
            gotInventoryId,
            alreadyRedirected: redirectedRef.current,
          })
        ) {
          redirectedRef.current = true;
          navigate(buildInventoryListCreatePath(gotInventoryId), { replace: true });
        }

        setDTO(data);

        if (!initializedPriceRowsRef.current) {
          const nextRows = initPriceRowsFromDTO(data);
          setPriceRows(nextRows);
          initializedPriceRowsRef.current = true;
        }
      } catch (e) {
        if (cancelled) return;
        const msg = String(e instanceof Error ? e.message : e);
        setDTOError(msg);
      } finally {
        if (cancelled) return;
        setLoadingDTO(false);
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [navigate, inventoryId, resolvedParams]);

  const { productBrandName, productName, tokenBrandName, tokenName } = React.useMemo(
    () => extractDisplayStrings(dto),
    [dto],
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

    listingTitle,
    setListingTitle,
    description,
    setDescription,

    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef,
    onSelectImages,
    onDropImages,
    onDragOverImages,
    removeImageAt,
    clearImages,

    assigneeName,
    assigneeCandidates: (assigneeCandidates ?? []) as Array<{ id: string; name: string }>,
    loadingMembers: Boolean(loadingMembers),
    handleSelectAssignee,

    decision,
    setDecision,
  };
}

