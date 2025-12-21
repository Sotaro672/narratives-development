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

export type UseListDetailResult = {
  pageTitle: string;
  onBack: () => void;

  // loading/error
  loading: boolean;
  error: string;

  // raw dto
  dto: ListDetailDTO | null;

  // =========================
  // ✅ Edit mode (page header)
  // =========================
  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;

  // ✅ listDetail.tsx が payload を渡してくるので受け取れる形にする
  // (payload は使えるものだけ使う / 未指定でも保存できる)
  onSave: (payload?: any) => Promise<void>;

  // ✅ 互換: onSaveEdit を探す UI があるため alias を用意
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

  // decision/status (view)
  decision: "list" | "hold" | "" | string;

  // ✅ display strings (already trimmed)
  productBrandId: string;
  productBrandName: string;
  productName: string;

  tokenBrandId: string;
  tokenBrandName: string;
  tokenName: string;

  // images (view) ※ edit は UI 側で切替（この hook では URL のみ返す）
  imageUrls: string[];
  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<React.SetStateAction<number>>;

  // price (PriceCard 用)
  // ✅ edit 中は draftPriceRows を返す
  priceRows: PriceRow[];
  draftPriceRows: PriceRow[];
  setDraftPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
  priceCard: ReturnType<typeof usePriceCard>;

  // ✅ admin (view) : assigneeId + assigneeName を返す
  assigneeId: string;
  assigneeName: string;

  createdByName: string;
  createdAt: string;
};

// ==============================
// local helpers (no duplication with backend helpers.go etc.)
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

function normalizePricesFromPriceRows(rows: PriceRow[]): Array<{ modelId: string; price: number }> {
  const out: Array<{ modelId: string; price: number }> = [];

  for (const r of Array.isArray(rows) ? rows : []) {
    const modelId = s((r as any)?.id); // ✅ PriceRow は id (= modelId) が正
    const price = toNumberOrNull((r as any)?.price);

    // modelId が空の行は送らない（UI側の暫定行など）
    if (!modelId) continue;

    // price が null の行は保存不可（backend が number 前提の可能性が高い）
    if (price === null) {
      throw new Error("missing_price_in_priceRows");
    }

    out.push({ modelId, price });
  }

  return out;
}

// decision -> backend status (best-effort)
function decisionToBackendStatus(decision: unknown): string | undefined {
  const d = s(decision).toLowerCase();
  if (!d) return undefined;
  if (d === "list") return "listing";
  if (d === "hold") return "hold";
  // それ以外は backend が受けられるならそのまま
  return d;
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

  // ============================================================
  // ✅ Edit state + drafts
  // ============================================================
  const [isEdit, setIsEdit] = React.useState(false);

  const [draftListingTitle, setDraftListingTitle] = React.useState(listingTitle);
  const [draftDescription, setDraftDescription] = React.useState(description);
  const [draftPriceRows, setDraftPriceRows] = React.useState<PriceRow[]>(priceRows);

  // DTO/derived が更新されたら、編集していない時だけ draft を同期
  React.useEffect(() => {
    if (isEdit) return;
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(priceRows);
  }, [isEdit, listingTitle, description, priceRows]);

  const onEdit = React.useCallback(() => {
    // enter edit: 最新の view 値で draft を初期化
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(priceRows);
    setIsEdit(true);
  }, [listingTitle, description, priceRows]);

  const onCancel = React.useCallback(() => {
    // cancel: view に巻き戻す
    setDraftListingTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(priceRows);
    setIsEdit(false);
  }, [listingTitle, description, priceRows]);

  // ============================================================
  // ✅ Save (PUT /lists/{id})
  // - ここで PUT を飛ばすので list_handler.go が確実に叩かれる
  // - 失敗時は edit を維持し、error を表示できるようにする
  // ============================================================
  const onSave = React.useCallback(
    async (payload?: any) => {
      const id = s(listId);
      if (!id) {
        setError("invalid_list_id");
        return;
      }

      // payload が渡ってくる場合は title/description/priceRows を尊重（ただし draft を優先）
      const nextTitle = s(draftListingTitle) || s(payload?.title) || s(payload?.listingTitle) || "";
      const nextDesc =
        s(draftDescription) ||
        s(payload?.description) ||
        s(payload?.detail?.description) ||
        "";

      // price rows: draft を正とする（UI側の編集結果）
      const nextPriceRows = Array.isArray(draftPriceRows) ? draftPriceRows : [];

      let prices: Array<{ modelId: string; price: number }> = [];
      try {
        prices = normalizePricesFromPriceRows(nextPriceRows);
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);
        setError(msg);
        return;
      }

      // ✅ backend status（既存 decision を維持する。payload に decision が来ても一応使える）
      const backendStatus =
        decisionToBackendStatus(payload?.decision) ||
        decisionToBackendStatus(decision) ||
        undefined;

      // ✅ updatedBy（auth.uid）
      const uid = s(auth.currentUser?.uid) || "system";

      // ✅ update payload（最小）
      // NOTE: backend が「domain List を丸ごと Unmarshal」する場合に備えて id/inventoryId/assigneeId も入れる
      const updatePayload: Record<string, any> = {
        id, // 互換: body id
        title: nextTitle,
        description: nextDesc,
        prices, // [{modelId, price}]
        status: backendStatus, // optional
        updatedBy: uid, // optional
      };

      // body に余計なものを入れない（DisallowUnknownFields 対策）
      if (s(dto?.inventoryId)) updatePayload.inventoryId = s(dto?.inventoryId);
      if (s(dto?.assigneeId)) updatePayload.assigneeId = s(dto?.assigneeId);

      // eslint-disable-next-line no-console
      console.log("[console/list/update] PUT start", {
        listId: id,
        titleLen: nextTitle.length,
        descLen: nextDesc.length,
        pricesCount: prices.length,
      });

      setLoading(true);
      setError("");

      try {
        const updated = await requestJSON<any>({
          method: "PUT",
          path: `/lists/${encodeURIComponent(id)}`,
          body: updatePayload,
        });

        // eslint-disable-next-line no-console
        console.log("[console/list/update] PUT ok", { listId: id });

        if (cancelledRef.current) return;

        // backend が detail dto を返す/返さない両方に耐える（best-effort）
        if (updated && typeof updated === "object") {
          setDTO(updated as ListDetailDTO);
        } else {
          // 返ってこないなら reload
          await reload();
        }

        setIsEdit(false);
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);

        // eslint-disable-next-line no-console
        console.log("[console/list/update] PUT failed", { listId: id, error: msg });

        if (cancelledRef.current) return;
        setError(msg);

        // edit は維持（ユーザが修正できるように）
      } finally {
        if (cancelledRef.current) return;
        setLoading(false);
      }
    },
    [
      listId,
      dto,
      decision,
      draftListingTitle,
      draftDescription,
      draftPriceRows,
      cancelledRef,
      reload,
    ],
  );

  // 互換 alias
  const onSaveEdit = onSave;

  // ============================================================
  // ✅ PriceCard hook（view/edit 切替）
  // - edit 中は draftPriceRows を rows に渡す
  // - edit 中は onChangePrice で draft を更新
  // ============================================================
  const handleChangePrice = React.useCallback(
    (index: number, price: number | null, row: PriceRow) => {
      setDraftPriceRows((prev) => {
        const next = Array.isArray(prev) ? [...prev] : [];
        if (!next[index]) return prev;

        // row 自体は信頼せず、index で更新（size/color/stock/rgb は維持）
        next[index] = {
          ...next[index],
          price,
          // 念のため既存 row の主要フィールドは維持（prev 側が正）
          size: next[index].size,
          color: next[index].color,
          rgb: (next[index] as any).rgb,
          stock: (next[index] as any).stock,
        };
        return next;
      });
    },
    [],
  );

  const effectivePriceRows = isEdit ? draftPriceRows : priceRows;

  const priceCard = usePriceCard({
    title: "価格",
    rows: effectivePriceRows,
    mode: isEdit ? "edit" : "view",
    currencySymbol: "¥",
    onChangePrice: isEdit ? handleChangePrice : undefined,
    // showTotal は既に削除済み前提（PriceCard 側も合計行無し）
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

    dto,

    // edit
    isEdit,
    onEdit,
    onCancel,
    onSave,
    onSaveEdit,

    // view
    listingTitle,
    description,

    // draft
    draftListingTitle,
    setDraftListingTitle,
    draftDescription,
    setDraftDescription,

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

    // price
    priceRows: effectivePriceRows,
    draftPriceRows,
    setDraftPriceRows,
    priceCard,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,
  };
}
