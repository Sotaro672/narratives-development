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
  openImagePicker: () => void;
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

  // ✅ null を含むのが正しい（useRef 初期値 null のため）
  const imageInputRef = React.useRef<HTMLInputElement | null>(null);

  const openImagePicker = React.useCallback(() => {
    imageInputRef.current?.click();
  }, []);

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

  const removeImageAt = React.useCallback(
    (idx: number) => {
      setImages((prev) => prev.filter((_, i) => i !== idx));

      setMainImageIndex((prevMain) => {
        if (idx === prevMain) return 0;
        if (idx < prevMain) return Math.max(0, prevMain - 1);
        return prevMain;
      });
    },
    [setImages],
  );

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
  // ✅ PriceRows（DTOから初期化し、以後はユーザー入力を保持）
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

  // ✅ PriceCard hook
  const priceCard = usePriceCard({
    title: "価格",
    rows: priceRows as unknown as PriceRow[], // usePriceCard は余分なフィールドを気にしない
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
  // ✅ 担当者選択（ID を保持して POST に渡せるようにする）
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
  // ✅ 作成（POST /lists） + ✅ Policy A: signedUrl で画像アップロード&登録（serviceへ移譲）
  // ============================================================
  const onCreate = React.useCallback(() => {
    void (async () => {
      try {
        // eslint-disable-next-line no-console
        console.log("[inventory/useListCreate] onCreate start", {
          inventoryId,
          productBlueprintId,
          tokenBlueprintId,
          decision,
          listingTitleLen: listingTitle.length,
          descriptionLen: description.length,
          imagesCount: images.length,
          mainImageIndex,
          priceRowsCount: priceRows.length,
          assigneeId,
        });

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
        // eslint-disable-next-line no-console
        console.warn("[inventory/useListCreate] onCreate failed", { msg, raw: e });
        alert(msg);
      }
    })();
  }, [
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    decision,
    listingTitle,
    description,
    images,
    mainImageIndex,
    priceRows,
    assigneeId,
    navigate,
    resolvedParams,
  ]);

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
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await loadListCreateDTOFromParams(resolvedParams);
        if (cancelled) return;

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
          navigate(buildInventoryListCreatePath(gotInventoryId), { replace: true });
        }

        setDTO(data);

        // ✅ priceRows 初期化（DTOの modelResolver 結果を PriceCard に渡す）
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

  // ✅ 表示文字列は service へ
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
    openImagePicker,
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
