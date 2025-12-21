// frontend/console/list/src/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

// ✅ PriceCard hook（PriceRow 型もここから取り込む）
import {
  usePriceCard,
  type PriceRow,
} from "../../../../list/src/presentation/hook/usePriceCard";

// Firebase Auth（IDトークン）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ API_BASE は repository の定義を正とする（base URL の名揺れ防止）
import { API_BASE } from "../../infrastructure/http/listRepositoryHTTP";

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

  // eslint-disable-next-line no-console
  console.log("[console/list/update] request", {
    method: args.method,
    url,
    bodyBytes: bodyText ? bodyText.length : 0,
    body: args.body,
    bodyJSON: bodyText,
  });

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

  // eslint-disable-next-line no-console
  console.log("[console/list/update] response", {
    method: args.method,
    url,
    status: res.status,
    ok: res.ok,
    responseBytes: text ? text.length : 0,
    json,
  });

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
    const modelId = s((r as any)?.id); // ✅ PriceRow は id (= modelId)
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

    createdByName,
    createdAt,
  } = derived;

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

  const [draftImages, setDraftImages] = React.useState<DraftImage[]>(
    cloneDraftImagesFromUrls(viewImageUrls),
  );

  // save state
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState("");

  // DTO/derived が更新されたら、編集していない時だけ draft を同期
  React.useEffect(() => {
    if (isEdit) return;

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
  }, [isEdit, listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls]);

  const onEdit = React.useCallback(() => {
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");

    // eslint-disable-next-line no-console
    console.log("[console/list/edit] enter", {
      listId: s(listId),
      listingTitle,
      descriptionLen: s(description).length,
      decision: decisionNorm,
      priceRowsCount: Array.isArray(viewPriceRows) ? viewPriceRows.length : 0,
    });

    setIsEdit(true);
  }, [listId, listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls]);

  const onCancel = React.useCallback(() => {
    // blob 解放
    revokeDraftBlobUrls(draftImages);

    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(clonePriceRows(viewPriceRows));
    setDraftDecision(decisionNorm);
    setDraftImages(cloneDraftImagesFromUrls(viewImageUrls));
    setSaveError("");

    setIsEdit(false);
  }, [draftImages, listingTitle, description, viewPriceRows, decisionNorm, viewImageUrls]);

  const onToggleDecision = React.useCallback(
    (next: ListingDecisionNorm) => {
      if (!isEdit) return;
      if (saving) return;
      setDraftDecision(next);
    },
    [isEdit, saving],
  );

  // ============================================================
  // ✅ images (UI-only)
  // ============================================================
  const onAddImages = React.useCallback(
    (files: FileList | null) => {
      if (!isEdit) return;
      if (!files || files.length === 0) return;

      const next: DraftImage[] = [];
      for (let i = 0; i < files.length; i++) {
        const f = files.item(i);
        if (!f) continue;
        const url = URL.createObjectURL(f);
        next.push({ url, file: f, isNew: true });
      }

      setDraftImages((prev) => [...(Array.isArray(prev) ? prev : []), ...next]);
    },
    [isEdit],
  );

  const onRemoveImageAt = React.useCallback(
    (idx: number) => {
      if (!isEdit) return;

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
    [isEdit],
  );

  // ============================================================
  // ✅ effective image urls (view/edit)
  // ============================================================
  const effectiveImageUrls = React.useMemo(() => {
    if (isEdit) {
      return (Array.isArray(draftImages) ? draftImages : [])
        .map((x) => s(x?.url))
        .filter(Boolean);
    }
    return (Array.isArray(viewImageUrls) ? viewImageUrls : []).map((u) => s(u)).filter(Boolean);
  }, [isEdit, draftImages, viewImageUrls]);

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

      // eslint-disable-next-line no-console
      console.log("[console/list/priceCard] onChangePrice", {
        listId: s(listId),
        index,
        nextPrice: price,
        rowSnapshot: {
          id: s((row as any)?.id),
          size: s((row as any)?.size),
          color: s((row as any)?.color),
          prevPrice: (row as any)?.price ?? null,
        },
      });

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
    [isEdit, listId],
  );

  // ============================================================
  // ✅ Save (PUT /lists/{id})
  //
  // ✅重要:
  // - backend 更新は `prices` が本命（listRepositoryHTTP と合わせる）
  // - PUT のレスポンスは信用せず、成功後に必ず GET で取り直して dto を更新
  // ============================================================
  const onSave = React.useCallback(
    async (payload?: any) => {
      const id = s(listId);
      if (!id) {
        setSaveError("invalid_list_id");
        return;
      }

      // ✅ タイトル/説明は payload があれば優先、無ければ draft を採用
      const nextTitle =
        s(payload?.title) ||
        s(payload?.listingTitle) ||
        s(draftListingTitle) ||
        "";

      const nextDesc =
        payload && payload.description !== undefined
          ? String(payload.description ?? "")
          : String(draftDescription ?? "");

      // ✅ 価格は draft が正
      const nextPriceRows = Array.isArray(draftPriceRows) ? draftPriceRows : [];

      let prices: Array<{ modelId: string; price: number }> = [];
      try {
        prices = normalizePricesFromPriceRows(nextPriceRows);
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);
        setSaveError(msg);
        return;
      }

      // ✅ status/decision は draftDecision を正とし、payload があれば吸収
      const backendStatus =
        decisionToBackendStatus(payload?.status) ||
        decisionToBackendStatus(payload?.decision) ||
        decisionToBackendStatus(draftDecision) ||
        decisionToBackendStatus(decisionNorm) ||
        undefined;

      const uid = s(auth.currentUser?.uid) || "system";

      const updatePayload: Record<string, any> = {
        id, // 互換: body id
        title: nextTitle,
        description: nextDesc,
        prices, // ✅ 本命
        status: backendStatus,
        updatedBy: uid,
      };

      // DisallowUnknownFields 対策で余計なものは入れない
      if (s(dto?.inventoryId)) updatePayload.inventoryId = s(dto?.inventoryId);
      if (s(dto?.assigneeId)) updatePayload.assigneeId = s(dto?.assigneeId);

      // eslint-disable-next-line no-console
      console.log("[console/list/update] PUT payload(final)", {
        listId: id,
        keys: Object.keys(updatePayload),
        status: backendStatus,
        pricesCount: Array.isArray(updatePayload.prices) ? updatePayload.prices.length : 0,
        pricesSample: (Array.isArray(updatePayload.prices) ? updatePayload.prices : []).slice(0, 4),
      });

      setSaving(true);
      setSaveError("");

      try {
        await requestJSON<any>({
          method: "PUT",
          path: `/lists/${encodeURIComponent(id)}`,
          body: updatePayload,
        });

        // ✅ PUT レスポンスは信用せず GET で取り直す
        const fresh = await loadListDetailDTO({
          listId: id,
          inventoryIdHint: inventoryId,
        });

        if (cancelledRef.current) return;

        // blob 解放（edit 終了で参照しなくなる）
        revokeDraftBlobUrls(draftImages);

        setDTO(fresh);
        setIsEdit(false);

        // eslint-disable-next-line no-console
        console.log("[console/list/update] after-save reload ok", {
          listId: id,
          freshPricesCount: Array.isArray((fresh as any)?.prices)
            ? (fresh as any).prices.length
            : undefined,
        });
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);

        // eslint-disable-next-line no-console
        console.log("[console/list/update] PUT failed", { listId: id, error: msg });

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
      draftImages,
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
    draftImages,
    onAddImages,
    onRemoveImageAt,

    mainImageIndex,
    setMainImageIndex,

    priceRows: viewPriceRows,
    draftPriceRows,
    setDraftPriceRows,
    onChangePrice,
    priceCard,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,
  };
}
