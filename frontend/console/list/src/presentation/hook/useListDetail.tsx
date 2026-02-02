// frontend/console/list/src/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ✅ PriceCard hook
import { usePriceCard } from "../../../../list/src/presentation/hook/usePriceCard";

// ✅ 型は inventory/application を正とする（依存方向を正す）
import type { PriceRow } from "../../../../inventory/src/application/listCreate/priceCard.types";

// Firebase Auth（IDトークン）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ API_BASE は repository の定義を正とする（base URL の名揺れ防止）
import { API_BASE } from "../../infrastructure/http/list";

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

export type ListingDecisionNorm = "listing" | "holding" | "";

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
  // ✅ Edit mode (page header)
  // =========================
  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;

  // ✅ listDetail.tsx が payload を渡してくるので受け取れる形にする（payload 無しでも動く）
  onSave: (payload?: any) => Promise<void>;
  onSaveEdit: (payload?: any) => Promise<void>;

  // =========================
  // listing (view/edit)
  // =========================
  listingTitle: string;
  description: string;

  // ✅ draft (edit UI 用)
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

  // 互換: 呼び出し側で使っている可能性があるため残す
  priceCard: ReturnType<typeof usePriceCard>;

  // admin (view)
  assigneeId: string;
  assigneeName: string;

  createdByName: string;
  createdAt: string;

  // ✅ NEW: 更新者/更新日時（listDetail.tsx 側で参照するため）
  updatedByName: string;
  updatedAt: string;
};

// ==============================
// local helpers
// ==============================

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) throw new Error("not_authenticated");
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "PUT" | "PATCH" | "POST" | "GET" | "DELETE";
  path: string;
  body?: unknown;
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  const bodyText = args.body === undefined ? undefined : JSON.stringify(args.body);

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: bodyText,
  });

  const text = await res.text();

  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }

  if (!res.ok) {
    const msg =
      (json && typeof json === "object" && (json.error || json.message)) ||
      `http_error_${res.status}`;
    throw new Error(String(msg));
  }

  return json as T;
}

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return n;
}

function normalizePricesFromPriceRows(
  rows: PriceRow[],
): Array<{ modelId: string; price: number }> {
  const out: Array<{ modelId: string; price: number }> = [];

  for (const r of Array.isArray(rows) ? rows : []) {
    const modelId = s((r as any)?.id); // ✅ PriceRow.id が modelId
    const price = toNumberOrNull((r as any)?.price);

    if (!modelId) continue;

    if (price === null) {
      throw new Error("missing_price_in_priceRows");
    }

    out.push({ modelId, price });
  }

  return out;
}

// ✅ 出品/保留の正規化（backend: listing/holding を想定、旧: list/hold も吸収）
function normalizeDecision(v: unknown): ListingDecisionNorm {
  const x = s(v).toLowerCase();
  if (x === "listing" || x === "list") return "listing";
  if (x === "holding" || x === "hold") return "holding";
  return "";
}

// decision/status -> backend status (best-effort)
function decisionToBackendStatus(v: unknown): string | undefined {
  const d = s(v).toLowerCase();
  if (!d) return undefined;

  // already normalized
  if (d === "listing" || d === "holding") return d;

  // legacy
  if (d === "list") return "listing";
  if (d === "hold") return "holding";

  return d;
}

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

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

// ✅ yyyy/mm/dd/hh/mm 形式（入力が不正ならそのまま返す）
function formatYMDHM(v: unknown): string {
  const raw = s(v);
  if (!raw) return "";

  const d = new Date(raw);
  if (!Number.isFinite(d.getTime())) return raw;

  const yyyy = d.getFullYear();
  const mm = pad2(d.getMonth() + 1);
  const dd = pad2(d.getDate());
  const hh = pad2(d.getHours());
  const mi = pad2(d.getMinutes());

  return `${yyyy}/${mm}/${dd}/${hh}/${mi}`;
}

// ==============================
// ✅ NEW: listImage を扱う hook（UI-only）
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

  const clearImages = React.useCallback(() => {
    if (!isEdit) return;
    if (saving) return;

    setDraftImages((prev) => {
      revokeDraftBlobUrls(prev);
      return [];
    });
  }, [isEdit, saving]);

  const imageUrls = React.useMemo(() => {
    return (Array.isArray(draftImages) ? draftImages : [])
      .map((x) => s(x?.url))
      .filter(Boolean);
  }, [draftImages]);

  return {
    draftImages,
    setDraftImages,
    imageUrls,
    onAddImages,
    onRemoveImageAt,
    clearImages,
    addFiles,
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

  // ✅ dto 側を最優先にして createdByName/createdAt を確定する
  const dtoAny: any = dto as any;

  const createdBy = s(dtoAny?.createdBy);
  const createdByNameFromDTO = s(dtoAny?.createdByName);
  const effectiveCreatedByName =
    createdByNameFromDTO || s(createdByNameFromDerived) || createdBy;

  const createdAtRaw = s(dtoAny?.createdAt) || s(createdAtRawFromDerived);

  // ✅ updated も dto を最優先で拾う（deriveListDetail 側の差分吸収）
  const updatedBy = s(dtoAny?.updatedBy);
  const updatedByNameFromDTO = s(dtoAny?.updatedByName);
  const updatedByNameFromDerived = s((derived as any)?.updatedByName);
  const effectiveUpdatedByName =
    updatedByNameFromDTO || updatedByNameFromDerived || updatedBy;

  const updatedAtRaw = s(dtoAny?.updatedAt) || s((derived as any)?.updatedAt);

  const createdAt = React.useMemo(() => formatYMDHM(createdAtRaw), [createdAtRaw]);
  const updatedAt = React.useMemo(() => formatYMDHM(updatedAtRaw), [updatedAtRaw]);

  const decisionNorm = React.useMemo(() => normalizeDecision(decision), [decision]);

  // ============================================================
  // ✅ Edit state + drafts
  // ============================================================
  const [isEdit, setIsEdit] = React.useState(false);

  const [draftListingTitle, setDraftListingTitle] = React.useState(listingTitle);
  const [draftDescription, setDraftDescription] = React.useState(description);

  const [draftPriceRows, setDraftPriceRows] = React.useState<PriceRow[]>(
    clonePriceRows(viewPriceRows),
  );

  const [draftDecision, setDraftDecision] = React.useState<ListingDecisionNorm>(
    decisionNorm,
  );

  // save state
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState("");

  // ============================================================
  // ✅ NEW: listImage hook（draftImages / add / remove / clear）
  // ============================================================
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
  }, [
    isEdit,
    listingTitle,
    description,
    viewPriceRows,
    decisionNorm,
    viewImageUrls,
    img,
  ]);

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
    // blob 解放
    revokeDraftBlobUrls(img.draftImages);

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    img.setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");

    setIsEdit(false);
  }, [
    img.draftImages,
    listingTitle,
    description,
    viewPriceRows,
    decisionNorm,
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

  // ============================================================
  // ✅ effective image urls (view/edit)
  // ============================================================
  const effectiveImageUrls = React.useMemo(() => {
    if (isEdit) return img.imageUrls;
    return (Array.isArray(viewImageUrls) ? viewImageUrls : [])
      .map((u) => s(u))
      .filter(Boolean);
  }, [isEdit, img.imageUrls, viewImageUrls]);

  // images: main index
  const [mainImageIndex, setMainImageIndex] = React.useState(0);
  useMainImageIndexGuard({
    imageUrls: effectiveImageUrls,
    mainImageIndex,
    setMainImageIndex,
  });

  // ============================================================
  // ✅ Price change (PriceCard -> draftPriceRows)
  // ============================================================
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

  // ============================================================
  // ✅ Save (PUT /lists/{id})
  // ============================================================
  const onSave = React.useCallback(
    async (payload?: any) => {
      const id = s(listId);
      if (!id) {
        setSaveError("invalid_list_id");
        return;
      }

      const nextTitle =
        s(payload?.title) ||
        s(payload?.listingTitle) ||
        s(draftListingTitle) ||
        "";

      const nextDesc =
        payload && payload.description !== undefined
          ? String(payload.description ?? "")
          : String(draftDescription ?? "");

      const nextPriceRows = Array.isArray(draftPriceRows) ? draftPriceRows : [];

      let prices: Array<{ modelId: string; price: number }> = [];
      try {
        prices = normalizePricesFromPriceRows(nextPriceRows);
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);
        setSaveError(msg);
        return;
      }

      const backendStatus =
        decisionToBackendStatus(payload?.status) ||
        decisionToBackendStatus(payload?.decision) ||
        decisionToBackendStatus(draftDecision) ||
        decisionToBackendStatus(decisionNorm) ||
        undefined;

      const uid = s(auth.currentUser?.uid) || "system";

      const updatePayload: Record<string, any> = {
        id,
        title: nextTitle,
        description: nextDesc,
        prices,
        status: backendStatus,
        updatedBy: uid,
      };

      if (s(dto?.inventoryId)) updatePayload.inventoryId = s((dto as any)?.inventoryId);
      if (s((dto as any)?.assigneeId)) updatePayload.assigneeId = s((dto as any)?.assigneeId);

      setSaving(true);
      setSaveError("");

      try {
        await requestJSON<any>({
          method: "PUT",
          path: `/lists/${encodeURIComponent(id)}`,
          body: updatePayload,
        });

        const fresh = await loadListDetailDTO({
          listId: id,
          inventoryIdHint: inventoryId,
        });

        if (cancelledRef.current) return;

        // blob 解放（draft 由来のみ）
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

  const onSaveEdit = onSave;

  // ============================================================
  // ✅ PriceCard hook（互換で残す）
  // ============================================================
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
    onSaveEdit,

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
