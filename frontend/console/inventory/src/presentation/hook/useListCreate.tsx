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
  type ImageInputRef,
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

  // ✅ NEW: listing local states (moved from page)
  listingTitle: string;
  setListingTitle: React.Dispatch<React.SetStateAction<string>>;
  description: string;
  setDescription: React.Dispatch<React.SetStateAction<string>>;

  // ✅ NEW: images (moved from page)
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

function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(prev.map((f) => `${f.name}__${f.size}__${f.lastModified}`));
  const filtered = add.filter((f) => !exists.has(`${f.name}__${f.size}__${f.lastModified}`));
  return [...prev, ...filtered];
}

// ✅ NEW: trim helper
function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ✅ NEW: price validation helper
function toNumberOrNaN(v: unknown): number {
  if (typeof v === "number") return v;
  if (typeof v === "string") {
    const n = Number(v.trim());
    return n;
  }
  return Number.NaN;
}

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
  // ✅ NEW: listing fields (moved from page)
  // ============================================================
  const [listingTitle, setListingTitle] = React.useState<string>("");
  const [description, setDescription] = React.useState<string>("");

  // ============================================================
  // ✅ NEW: images (moved from page)
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

    setImages((prev) => {
      const next = dedupeFiles(prev, files);
      return next;
    });

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

  // ✅ NEW: validation
  const validateBeforeCreate = React.useCallback(() => {
    const titleTrim = s(listingTitle);
    if (!titleTrim) {
      throw new Error("タイトルを入力してください。");
    }

    if (!Array.isArray(priceRows) || priceRows.length === 0) {
      throw new Error("価格が未設定です（価格行がありません）。");
    }

    // price が 0 / null / NaN の行が1つでもあれば NG
    const bad = priceRows.find((r) => {
      const p = (r as any)?.price;
      // null/undefined -> NG
      if (p === null || p === undefined) return true;
      const n = toNumberOrNaN(p);
      // NaN -> NG / 0 or less -> NG（要件: price:0 でエラー）
      if (!Number.isFinite(n)) return true;
      if (n <= 0) return true;
      return false;
    });

    if (bad) {
      throw new Error("価格が未入力、または 0 円の行があります。各行の価格を 1 円以上に設定してください。");
    }
  }, [listingTitle, priceRows]);

  const onCreate = React.useCallback(() => {
    try {
      validateBeforeCreate();

      // eslint-disable-next-line no-console
      console.log("[inventory/useListCreate] onCreate validated", {
        inventoryId,
        productBlueprintId,
        tokenBlueprintId,
        decision,
        listingTitle,
        descriptionLen: description.length,
        imagesCount: images.length,
        priceRowsCount: priceRows.length,
        priceRowsSample: priceRows.slice(0, 5),
      });

      alert("作成しました（仮）");
      navigate(buildAfterCreatePath(resolvedParams));
    } catch (e) {
      const msg = String(e instanceof Error ? e.message : e);
      // eslint-disable-next-line no-console
      console.warn("[inventory/useListCreate] onCreate validation failed", { msg, raw: e });
      alert(msg);
    }
  }, [
    validateBeforeCreate,
    navigate,
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    decision,
    listingTitle,
    description,
    images,
    priceRows,
    resolvedParams,
  ]);

  // eslint-disable-next-line no-console
  React.useEffect(() => {
    console.log("[inventory/useListCreate] create snapshot", {
      decision,
      priceRowsCount: priceRows.length,
      listingTitleLen: listingTitle.length,
      descriptionLen: description.length,
      imagesCount: images.length,
    });
  }, [decision, priceRows, listingTitle, description, images]);

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
      sample: Array.isArray(assigneeCandidates) ? assigneeCandidates.slice(0, 5) : [],
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

    listingTitle,
    setListingTitle,
    description,
    setDescription,

    images,
    imagePreviewUrls,
    mainImageIndex,
    setMainImageIndex,
    imageInputRef, // ✅ ここで型を value として使わない
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
