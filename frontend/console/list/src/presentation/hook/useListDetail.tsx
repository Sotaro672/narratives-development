// frontend/console/list/src/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ✅ PriceCard hook
import { usePriceCard } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ 型は inventory/application を正とする（依存方向を正す）
import type { PriceRow } from "../../../../inventory/src/application/listCreate/priceCard.types";

// Firebase Auth（uid 取得）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ internal hooks（presentation 層内で完結）
import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";

// ✅ DELETE API（画像削除）
import { deleteListImageHTTP } from "../../infrastructure/http/list";

// ✅ それ以外は service へ
import {
  resolveListDetailParams,
  loadListDetailDTO,
  updateListDetailDTO,
  deriveListDetail,
  computeListDetailPageTitle,
  normalizeListingDecisionNorm,
  toDecisionForUpdate,
  formatYMDHM,
  type ListingDecisionNorm,
  type ListDetailRouteParams,
  type ListDetailDTO,
  s,
  // ✅ 変更：保存前後差分は「確実に存在する imageUrls」を正にする
  normalizeImageUrls,
} from "../../application/listDetailService";

export type { ListingDecisionNorm };

export type DraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;

  // loading/error (load)
  loading: boolean;
  error: string;

  // save state (save)
  saving: boolean;
  saveError: string;

  // raw dto
  dto: ListDetailDTO | null;

  // reload (optional but handy)
  reload: () => Promise<void>;

  // =========================
  // Edit mode (page header)
  // =========================
  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;

  // listDetail.tsx が payload を渡してくるので受け取れる形にする（payload 無しでも動く）
  onSave: (payload?: any) => Promise<void>;

  // =========================
  // listing (view/edit)
  // =========================
  listingTitle: string;
  description: string;

  // draft (edit UI 用)
  draftListingTitle: string;
  setDraftListingTitle: React.Dispatch<React.SetStateAction<string>>;
  draftDescription: string;
  setDraftDescription: React.Dispatch<React.SetStateAction<string>>;

  // =========================
  // decision/status (view/edit)
  // =========================
  decision: "list" | "hold" | "" | string; // raw(view)
  decisionNorm: ListingDecisionNorm; // normalized(view)
  draftDecision: ListingDecisionNorm; // normalized(edit)
  setDraftDecision: React.Dispatch<React.SetStateAction<ListingDecisionNorm>>;
  onToggleDecision: (next: ListingDecisionNorm) => void;

  // =========================
  // display strings (already trimmed)
  // =========================
  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  // =========================
  // images (view/edit)
  // =========================
  imageUrls: string[]; // effective (view or edit)
  draftImages: DraftImage[]; // edit 用（UI-only）
  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (idx: number) => void;

  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  // =========================
  // price (PriceCard 用)
  // =========================
  priceRows: PriceRow[]; // view
  draftPriceRows: PriceRow[]; // edit
  setDraftPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  onChangePrice: (index: number, price: number | null, row: PriceRow) => void;

  // ✅ PriceCard result（page が参照するため）
  priceCard: ReturnType<typeof usePriceCard>;

  // admin (view)
  assigneeId: string;
  assigneeName: string;

  createdByName: string;
  createdAt: string;

  updatedByName: string;
  updatedAt: string;
};

// ==============================
// local helpers（UI-only）
// ==============================

function clonePriceRows(rows: PriceRow[]): PriceRow[] {
  return Array.isArray(rows) ? rows.map((x) => ({ ...(x as any) })) : [];
}

function cloneDraftImagesFromUrls(urls: string[]): DraftImage[] {
  return (Array.isArray(urls) ? urls : [])
    .map((u) => s(u))
    .filter(Boolean)
    .map((u) => ({ url: u, isNew: false as const }));
}

function revokeDraftBlobUrls(items: DraftImage[]) {
  for (const x of Array.isArray(items) ? items : []) {
    if (x?.isNew && typeof x?.url === "string" && x.url.startsWith("blob:")) {
      try {
        URL.revokeObjectURL(x.url);
      } catch {
        // noop
      }
    }
  }
}

// ==============================
// listImage draft hook（UI-only）
// ==============================

function fileKey(f: File): string {
  return `${f.name}__${f.size}__${f.lastModified}`;
}

function isImageFile(f: File): boolean {
  return String((f as any)?.type ?? "").startsWith("image/");
}

function useListImages(args: { isEdit: boolean; saving: boolean; initialUrls: string[] }) {
  const { isEdit, saving, initialUrls } = args;

  const [draftImages, setDraftImages] = React.useState<DraftImage[]>(
    cloneDraftImagesFromUrls(initialUrls),
  );

  React.useEffect(() => {
    if (isEdit) return;
    setDraftImages(cloneDraftImagesFromUrls(initialUrls));
  }, [isEdit, initialUrls]);

  const addFiles = React.useCallback(
    (files: File[]) => {
      if (!isEdit) return;
      if (saving) return;

      const incoming = (Array.isArray(files) ? files : []).filter(Boolean).filter(isImageFile);

      if (incoming.length === 0) return;

      setDraftImages((prev) => {
        const prevArr = Array.isArray(prev) ? prev : [];
        const exists = new Set(
          prevArr.filter((x) => x?.isNew && x?.file).map((x) => fileKey(x.file as File)),
        );

        const next: DraftImage[] = [];

        for (const f of incoming) {
          const k = fileKey(f);
          if (exists.has(k)) continue;
          exists.add(k);

          const url = URL.createObjectURL(f);
          next.push({ url, file: f, isNew: true });
        }

        return [...prevArr, ...next];
      });
    },
    [isEdit, saving],
  );

  const onAddImages = React.useCallback(
    (files: FileList | null) => {
      if (!files || files.length === 0) return;
      const arr = Array.from(files).filter(Boolean) as File[];
      addFiles(arr);
    },
    [addFiles],
  );

  const onRemoveImageAt = React.useCallback(
    (idx: number) => {
      if (!isEdit) return;
      if (saving) return;

      setDraftImages((prev) => {
        const arr = Array.isArray(prev) ? prev : [];
        if (idx < 0 || idx >= arr.length) return arr;

        const target = arr[idx];
        if (target?.isNew && target?.url?.startsWith("blob:")) {
          try {
            URL.revokeObjectURL(target.url);
          } catch {
            // noop
          }
        }

        return arr.slice(0, idx).concat(arr.slice(idx + 1));
      });
    },
    [isEdit, saving],
  );

  const imageUrls = React.useMemo(() => {
    return (Array.isArray(draftImages) ? draftImages : []).map((x) => s(x?.url)).filter(Boolean);
  }, [draftImages]);

  return {
    draftImages,
    setDraftImages,
    imageUrls,
    onAddImages,
    onRemoveImageAt,
  };
}

export function useListDetail(): UseListDetailResult {
  const navigate = useNavigate();
  const params = useParams<ListDetailRouteParams>();

  const resolved = React.useMemo(() => resolveListDetailParams(params), [params]);
  const { listId, inventoryId } = resolved;

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // -----------------------------
  // Load DTO
  // -----------------------------
  const [dto, setDTO] = React.useState<ListDetailDTO | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState("");

  const cancelledRef = useCancelledRef();

  const reload = React.useCallback(async () => {
    const id = s(listId);
    if (!id) {
      setDTO(null);
      setError("listId がありません（ルートパラメータを確認してください）。");
      return;
    }

    setLoading(true);
    setError("");

    try {
      const data = await loadListDetailDTO({
        listId: id,
        inventoryIdHint: inventoryId,
      });
      if (cancelledRef.current) return;

      setDTO(data);
    } catch (e) {
      if (cancelledRef.current) return;
      const msg = String(e instanceof Error ? e.message : e);
      setError(msg);
      setDTO(null);
    } finally {
      if (cancelledRef.current) return;
      setLoading(false);
    }
  }, [listId, inventoryId, cancelledRef]);

  React.useEffect(() => {
    void reload();
  }, [reload]);

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

    imageUrls: viewImageUrls,
    priceRows: viewPriceRows,

    assigneeId,
    assigneeName,

    createdByName: createdByNameFromDerived,
    createdAt: createdAtRawFromDerived,
  } = derived;

  // dto 優先
  const dtoAny: any = dto as any;

  const createdBy = s(dtoAny?.createdBy);
  const createdByNameFromDTO = s(dtoAny?.createdByName);
  const effectiveCreatedByName = createdByNameFromDTO || s(createdByNameFromDerived) || createdBy;

  const createdAtRaw = s(dtoAny?.createdAt) || s(createdAtRawFromDerived);

  const updatedBy = s(dtoAny?.updatedBy);
  const updatedByNameFromDTO = s(dtoAny?.updatedByName);
  const updatedByNameFromDerived = s((derived as any)?.updatedByName);
  const effectiveUpdatedByName = updatedByNameFromDTO || updatedByNameFromDerived || updatedBy;

  const updatedAtRaw = s(dtoAny?.updatedAt) || s((derived as any)?.updatedAt);

  // ✅ (1) moved to service
  const createdAt = React.useMemo(() => formatYMDHM(createdAtRaw), [createdAtRaw]);
  const updatedAt = React.useMemo(() => formatYMDHM(updatedAtRaw), [updatedAtRaw]);

  // ✅ (2) moved to service
  const decisionNorm = React.useMemo(() => normalizeListingDecisionNorm(decision), [decision]);

  // ============================================================
  // Edit state + drafts
  // ============================================================
  const [isEdit, setIsEdit] = React.useState(false);

  const [draftListingTitle, setDraftListingTitle] = React.useState(listingTitle);
  const [draftDescription, setDraftDescription] = React.useState(description);

  const [draftPriceRows, setDraftPriceRows] = React.useState<PriceRow[]>(clonePriceRows(viewPriceRows));

  const [draftDecision, setDraftDecision] = React.useState<ListingDecisionNorm>(decisionNorm);

  // save state
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState("");

  // images draft
  const img = useListImages({
    isEdit,
    saving,
    initialUrls: viewImageUrls,
  });

  // DTO/derived が更新されたら、編集していない時だけ draft を同期
  React.useEffect(() => {
    if (isEdit) return;

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);

    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
  }, [isEdit, listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls, img]);

  const onEdit = React.useCallback(() => {
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");
    setIsEdit(true);
  }, [listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls, img]);

  const onCancel = React.useCallback(() => {
    revokeDraftBlobUrls(img.draftImages);

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");

    setIsEdit(false);
  }, [img.draftImages, listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls, img]);

  const onToggleDecision = React.useCallback(
    (next: ListingDecisionNorm) => {
      if (!isEdit) return;
      if (saving) return;
      setDraftDecision(next);
    },
    [isEdit, saving],
  );

  // effective urls (view/edit)
  const effectiveImageUrls = React.useMemo(() => {
    if (isEdit) return img.imageUrls;
    return (Array.isArray(viewImageUrls) ? viewImageUrls : []).map((u) => s(u)).filter(Boolean);
  }, [isEdit, img.imageUrls, viewImageUrls]);

  // images: main index
  const [mainImageIndex, setMainImageIndex] = React.useState(0);
  useMainImageIndexGuard({
    imageUrls: effectiveImageUrls,
    mainImageIndex,
    setMainImageIndex,
  });

  // Price change -> draftPriceRows
  const onChangePrice = React.useCallback(
    (index: number, price: number | null, row: PriceRow) => {
      if (!isEdit) return;

      setDraftPriceRows((prev) => {
        const next = Array.isArray(prev) ? [...prev] : [];
        if (!next[index]) return prev;

        next[index] = {
          ...next[index],
          price,
          size: next[index].size,
          color: next[index].color,
          rgb: (next[index] as any).rgb,
          stock: (next[index] as any).stock,
        };

        return next;
      });
    },
    [isEdit],
  );

  // Save -> application service only
  const onSave = React.useCallback(
    async (payload?: any) => {
      const id = s(listId);
      if (!id) {
        setSaveError("invalid_list_id");
        return;
      }

      const nextTitle = s(payload?.title) || s(payload?.listingTitle) || s(draftListingTitle) || "";

      const nextDesc =
        payload && payload.description !== undefined
          ? String(payload.description ?? "")
          : String(draftDescription ?? "");

      // ✅ (2) moved to service
      const nextDecision =
        toDecisionForUpdate(payload?.decision) ||
        toDecisionForUpdate(payload?.status) ||
        toDecisionForUpdate(draftDecision) ||
        toDecisionForUpdate(decisionNorm) ||
        undefined;

      const uid = s(auth.currentUser?.uid) || "system";

      setSaving(true);
      setSaveError("");

      try {
        // ============================================================
        // ✅ 画像削除の差分反映（2枚→1枚 等）
        // - before は listImages に依存せず、確実に返る imageUrls を正とする
        // ============================================================
        const beforeUrls = normalizeImageUrls(dto);
        const afterUrls = (Array.isArray(img.draftImages) ? img.draftImages : [])
          .filter((x) => !x?.isNew) // 既存のみ
          .map((x) => s(x?.url))
          .filter(Boolean);

        const removedUrls = beforeUrls.filter((u) => !afterUrls.includes(u));

        // eslint-disable-next-line no-console
        console.log("[listImage] diff", {
          listId: id,
          before: beforeUrls,
          after: afterUrls,
          removed: removedUrls,
        });

        for (const u of removedUrls) {
          const imageIdOrObjOrUrl = s(u);
          if (!imageIdOrObjOrUrl) continue;

          // listApi 側が objectPath/URL から imageId 抽出して DELETE できる想定
          await deleteListImageHTTP({ listId: id, imageId: imageIdOrObjOrUrl });
        }

        // ============================================================
        // list 本体の更新
        // ============================================================
        await updateListDetailDTO({
          listId: id,
          title: nextTitle,
          description: nextDesc,

          // ✅ prices ではなく priceRows を渡す（repository 正規化前提）
          priceRows: Array.isArray(draftPriceRows) ? draftPriceRows : [],

          decision: nextDecision,
          assigneeId: s((dto as any)?.assigneeId) || undefined,
          updatedBy: uid,
        });

        const fresh = await loadListDetailDTO({
          listId: id,
          inventoryIdHint: inventoryId,
        });

        if (cancelledRef.current) return;

        revokeDraftBlobUrls(img.draftImages);

        setDTO(fresh);
        setIsEdit(false);
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);
        if (cancelledRef.current) return;
        setSaveError(msg);
      } finally {
        if (cancelledRef.current) return;
        setSaving(false);
      }
    },
    [
      listId,
      inventoryId,
      dto,
      decisionNorm,
      draftDecision,
      draftListingTitle,
      draftDescription,
      draftPriceRows,
      img.draftImages,
      cancelledRef,
    ],
  );

  // PriceCard
  const effectiveForPriceCard = isEdit ? draftPriceRows : viewPriceRows;

  const priceCard = usePriceCard({
    title: "価格",
    rows: effectiveForPriceCard,
    mode: isEdit ? "edit" : "view",
    currencySymbol: "¥",
    onChangePrice: isEdit ? onChangePrice : undefined,
  });

  const pageTitle = React.useMemo(
    () => computeListDetailPageTitle({ listId, listingTitle }),
    [listId, listingTitle],
  );

  return {
    pageTitle,
    onBack,

    loading,
    error,

    saving,
    saveError,

    dto,
    reload,

    isEdit,
    onEdit,
    onCancel,
    onSave,

    listingTitle,
    description,

    draftListingTitle,
    setDraftListingTitle,
    draftDescription,
    setDraftDescription,

    decision,
    decisionNorm,
    draftDecision,
    setDraftDecision,
    onToggleDecision,

    productBrandId,
    productBrandName,
    productName,

    tokenBrandId,
    tokenBrandName,
    tokenName,

    imageUrls: effectiveImageUrls,
    draftImages: img.draftImages,
    onAddImages: img.onAddImages,
    onRemoveImageAt: img.onRemoveImageAt,

    mainImageIndex,
    setMainImageIndex,

    priceRows: viewPriceRows,
    draftPriceRows,
    setDraftPriceRows,
    onChangePrice,

    priceCard,

    assigneeId,
    assigneeName,

    createdByName: effectiveCreatedByName,
    createdAt,

    updatedByName: effectiveUpdatedByName,
    updatedAt,
  };
}
