// frontend/console/list/src/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ✅ PriceCard hook
import { usePriceCard } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ 型は inventory/application を正とする（依存方向を正す）
import type { PriceRow } from "../../../../inventory/src/application/listCreate/listCreate.types";

// Firebase Auth
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ internal hooks（presentation 層内で完結）
import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";

import { saveListDetailChanges } from "../../application/listDetail/listDetailSave.usecase";

// ✅ それ以外は service へ
import {
  resolveListDetailParams,
  loadListDetailDTO,
  deriveListDetail,
  computeListDetailPageTitle,
  normalizeListingDecisionNorm,
  toDecisionForUpdate,
  formatYMDHM,
  type ListingDecisionNorm,
  type ListDetailRouteParams,
  type ListDetailDTO,
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
  decision: "list" | "hold" | "" | string;
  decisionNorm: ListingDecisionNorm;
  draftDecision: ListingDecisionNorm;
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
  imageUrls: string[];
  draftImages: DraftImage[];
  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (idx: number) => void;
  onClearImages: () => void;

  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  // =========================
  // price (PriceCard 用)
  // =========================
  priceRows: PriceRow[];
  draftPriceRows: PriceRow[];
  setDraftPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  onChangePrice: (index: number, price: number | null, row: PriceRow) => void;

  // ✅ PriceCard result（page が参照するため）
  priceCard: ReturnType<typeof usePriceCard>;

  // =========================
  // admin (view/edit)
  // =========================
  assigneeId: string;
  assigneeName: string;
  draftAssigneeId: string;
  setDraftAssigneeId: React.Dispatch<React.SetStateAction<string>>;
  onSelectAssignee: (id: string) => void;
  onChangeAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;

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
    .map((url) => String(url ?? "").trim())
    .filter(Boolean)
    .map((url) => ({ url, isNew: false as const }));
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

function useListImages(args: {
  isEdit: boolean;
  saving: boolean;
  initialUrls: string[];
}) {
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

      const incoming = (Array.isArray(files) ? files : [])
        .filter(Boolean)
        .filter(isImageFile);

      if (incoming.length === 0) return;

      setDraftImages((prev) => {
        const prevArr = Array.isArray(prev) ? prev : [];
        const exists = new Set(
          prevArr
            .filter((x) => x?.isNew && x?.file)
            .map((x) => fileKey(x.file as File)),
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

  const onClearImages = React.useCallback(() => {
    if (!isEdit) return;
    if (saving) return;

    setDraftImages((prev) => {
      const arr = Array.isArray(prev) ? prev : [];

      for (const x of arr) {
        if (x?.isNew && typeof x?.url === "string" && x.url.startsWith("blob:")) {
          try {
            URL.revokeObjectURL(x.url);
          } catch {
            // noop
          }
        }
      }

      return [];
    });
  }, [isEdit, saving]);

  const imageUrls = React.useMemo(() => {
    return (Array.isArray(draftImages) ? draftImages : [])
      .map((x) => String(x?.url ?? "").trim())
      .filter(Boolean);
  }, [draftImages]);

  return {
    draftImages,
    setDraftImages,
    imageUrls,
    onAddImages,
    onRemoveImageAt,
    onClearImages,
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
    const id = String(listId ?? "").trim();
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

  const createdBy = String(dtoAny?.createdBy ?? "").trim();
  const createdByNameFromDTO = String(dtoAny?.createdByName ?? "").trim();
  const effectiveCreatedByName =
    createdByNameFromDTO ||
    String(createdByNameFromDerived ?? "").trim() ||
    createdBy;

  const createdAtRaw =
    String(dtoAny?.createdAt ?? "").trim() ||
    String(createdAtRawFromDerived ?? "").trim();

  const updatedBy = String(dtoAny?.updatedBy ?? "").trim();
  const updatedByNameFromDTO = String(dtoAny?.updatedByName ?? "").trim();
  const updatedByNameFromDerived = String((derived as any)?.updatedByName ?? "").trim();
  const effectiveUpdatedByName =
    updatedByNameFromDTO || updatedByNameFromDerived || updatedBy;

  const updatedAtRaw =
    String(dtoAny?.updatedAt ?? "").trim() ||
    String((derived as any)?.updatedAt ?? "").trim();

  const createdAt = React.useMemo(
    () => formatYMDHM(createdAtRaw),
    [createdAtRaw],
  );

  const updatedAt = React.useMemo(
    () => formatYMDHM(updatedAtRaw),
    [updatedAtRaw],
  );

  const decisionNorm = React.useMemo(
    () => normalizeListingDecisionNorm(decision),
    [decision],
  );

  // ============================================================
  // Edit state + drafts
  // ============================================================
  const [isEdit, setIsEdit] = React.useState(false);

  const [draftListingTitle, setDraftListingTitle] =
    React.useState(listingTitle);
  const [draftDescription, setDraftDescription] =
    React.useState(description);

  const [draftPriceRows, setDraftPriceRows] = React.useState<PriceRow[]>(
    clonePriceRows(viewPriceRows),
  );

  const [draftDecision, setDraftDecision] =
    React.useState<ListingDecisionNorm>(decisionNorm);

  const [draftAssigneeId, setDraftAssigneeId] =
    React.useState(assigneeId);

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
    setDraftAssigneeId(assigneeId);

    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
  }, [
    isEdit,
    listingTitle,
    description,
    viewPriceRows,
    decisionNorm,
    assigneeId,
    viewImageUrls,
    img,
  ]);

  const onEdit = React.useCallback(() => {
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    setDraftAssigneeId(assigneeId);
    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");
    setIsEdit(true);
  }, [
    listingTitle,
    description,
    viewPriceRows,
    decisionNorm,
    assigneeId,
    viewImageUrls,
    img,
  ]);

  const onCancel = React.useCallback(() => {
    revokeDraftBlobUrls(img.draftImages);

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    setDraftAssigneeId(assigneeId);
    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");

    setIsEdit(false);
  }, [
    img.draftImages,
    listingTitle,
    description,
    viewPriceRows,
    decisionNorm,
    assigneeId,
    viewImageUrls,
    img,
  ]);

  const onToggleDecision = React.useCallback(
    (next: ListingDecisionNorm) => {
      if (!isEdit) return;
      if (saving) return;
      setDraftDecision(next);
    },
    [isEdit, saving],
  );

  const onSelectAssignee = React.useCallback(
    (id: string) => {
      if (!isEdit) return;
      if (saving) return;

      setDraftAssigneeId(String(id ?? "").trim());
    },
    [isEdit, saving],
  );

  const onChangeAssignee = React.useCallback(
    (id: string) => {
      if (!isEdit) return;
      if (saving) return;

      setDraftAssigneeId(String(id ?? "").trim());
    },
    [isEdit, saving],
  );

  const onEditAssignee = React.useCallback(() => {
    // AdminCard 側の編集導線用。
    // ListDetail 全体の edit mode で制御しているため、現状は no-op。
  }, []);

  const onClickAssignee = React.useCallback(() => {
    // 担当者クリック時の導線用。
    // 遷移先やモーダルが決まったらここに処理を追加する。
  }, []);

  // effective urls (view/edit)
  const effectiveImageUrls = React.useMemo(() => {
    if (isEdit) return img.imageUrls;

    return (Array.isArray(viewImageUrls) ? viewImageUrls : [])
      .map((url) => String(url ?? "").trim())
      .filter(Boolean);
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
        const src = Array.isArray(prev) ? prev : [];

        return src.map((item, i) => {
          if (i !== index) return item;

          return {
            ...(item as any),
            price,
          } as PriceRow;
        });
      });
    },
    [isEdit],
  );

  // Save -> application usecase
  const onSave = React.useCallback(
    async (payload?: any) => {
      const id = String(listId ?? "").trim();
      if (!id) {
        setSaveError("invalid_list_id");
        return;
      }

      const nextTitle =
        String(payload?.title ?? "").trim() ||
        String(payload?.listingTitle ?? "").trim() ||
        String(draftListingTitle ?? "").trim() ||
        "";

      const nextDesc =
        payload && payload.description !== undefined
          ? String(payload.description ?? "")
          : String(draftDescription ?? "");

      const nextDecision =
        toDecisionForUpdate(payload?.decision) ||
        toDecisionForUpdate(payload?.status) ||
        toDecisionForUpdate(draftDecision) ||
        toDecisionForUpdate(decisionNorm) ||
        undefined;

      const uid = String(auth.currentUser?.uid ?? "").trim() || "system";

      setSaving(true);
      setSaveError("");

      try {
        const result = await saveListDetailChanges({
          listId: id,
          inventoryIdHint: inventoryId,
          currentDTO: dto,

          title: nextTitle,
          description: nextDesc,
          decision: nextDecision,

          assigneeId:
            String(payload?.assigneeId ?? "").trim() ||
            String(draftAssigneeId ?? "").trim() ||
            String((dto as any)?.assigneeId ?? "").trim() ||
            undefined,
          updatedBy: uid,

          draftPriceRows: Array.isArray(draftPriceRows) ? draftPriceRows : [],
          draftImages: Array.isArray(img.draftImages) ? img.draftImages : [],

          mainImageIndex,
        });

        if (cancelledRef.current) return;

        revokeDraftBlobUrls(img.draftImages);

        setDTO(result.dto);
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
      draftAssigneeId,
      draftPriceRows,
      img.draftImages,
      mainImageIndex,
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
    onClearImages: img.onClearImages,

    mainImageIndex,
    setMainImageIndex,

    priceRows: viewPriceRows,
    draftPriceRows,
    setDraftPriceRows,
    onChangePrice,

    priceCard,

    assigneeId,
    assigneeName,
    draftAssigneeId,
    setDraftAssigneeId,
    onSelectAssignee,
    onChangeAssignee,
    onEditAssignee,
    onClickAssignee,

    createdByName: effectiveCreatedByName,
    createdAt,

    updatedByName: effectiveUpdatedByName,
    updatedAt,
  };
}