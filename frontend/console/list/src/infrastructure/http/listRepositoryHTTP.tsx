// frontend\console\list\src\infrastructure\http\listRepositoryHTTP.tsx
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend base URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

/**
 * ✅ ListImage bucket (public access想定: https://storage.googleapis.com/{bucket}/{objectPath})
 * - backend 側の fallback と合わせる
 * - 将来 env を増やす場合に備えて VITE_LIST_IMAGE_BUCKET も見ておく
 */
const ENV_LIST_IMAGE_BUCKET = String(
  (import.meta as any).env?.VITE_LIST_IMAGE_BUCKET ?? "",
).trim();

const FALLBACK_LIST_IMAGE_BUCKET = "narratives-development-list";

export const LIST_IMAGE_BUCKET = ENV_LIST_IMAGE_BUCKET || FALLBACK_LIST_IMAGE_BUCKET;

/**
 * ===========
 * Types
 * ===========
 * ※ backend の List エンティティに完全一致しなくてもOK（必要な分だけ）
 */

export type CreateListInput = {
  // backend が docId を要求する場合に備えて（未指定なら inventoryId を採用）
  id?: string;

  // ルート（inventory/list/create から作成する想定）
  // ✅ 方針A: inventoryId は「pb__tb」をそのまま通す（絶対に split しない）
  inventoryId?: string;

  // UI 入力
  title: string;
  description: string;

  // PriceCard の rows（UI 側は保持していてOK / backend には modelId + price のみ送る）
  priceRows?: Array<{
    modelId?: string;
    price: number | null;

    // UI 用（backend には送らない）
    size: string;
    color: string;
    stock: number;
    rgb?: number | null;
  }>;

  // 画面の「出品｜保留」（※ create payload には送らない）
  decision?: "list" | "hold";

  // 担当者など（必要に応じて）
  assigneeId?: string;

  // 作成者など（バックエンドで auth から取るなら省略可）
  createdBy?: string;
};

/**
 * ✅ NEW: Update 用
 * - listDetail 側の PriceRow は id = modelId なので、row.id も受ける
 * - backend に送るのは modelId + price だけ（DisallowUnknownFields 対策）
 */
export type UpdateListInput = {
  listId: string;

  title?: string;
  description?: string;

  // detail 側の priceRows（id=modelId）
  priceRows?: Array<{
    // create系: modelId
    modelId?: string;

    // detail系: id (= modelId)
    id?: string;

    price: number | null;

    // UI 用（backend には送らない）
    size?: string;
    color?: string;
    stock?: number;
    rgb?: number | null;
  }>;

  // UI の "list" | "hold" を backend の status に変換して送る（必要な場合のみ）
  decision?: "list" | "hold";

  assigneeId?: string;

  // バックエンドで auth から取るなら省略可
  updatedBy?: string;
};

export type ListDTO = Record<string, any>;
export type ListAggregateDTO = Record<string, any>;

/**
 * ✅ ListImage DTO（backend 依存を避けるため Record<any> を基本にする）
 * - ただし bucket/objectPath/publicUrl 等、URL 生成に必要なキーは代表的な候補を吸収する
 */
export type ListImageDTO = Record<string, any>;

/**
 * ✅ Signed URL 発行の戻り（Policy A）
 *
 * IMPORTANT:
 * - backend の返却は uploadUrl/publicUrl/objectPath/id... になりがち
 * - UI/呼び出し側は signedUrl を使いたいケースがあるため、この DTO では signedUrl を正とし、
 *   issueListImageSignedUrlHTTP 内で uploadUrl → signedUrl に正規化する
 */
export type SignedListImageUploadDTO = {
  id?: string;

  bucket?: string;

  // ✅ 必須（= GCS 上の objectPath / key）
  objectPath: string;

  // ✅ PUT 先（= signed URL）
  signedUrl: string;

  // ✅ 表示用（public access を想定）
  publicUrl?: string;

  // optional metadata
  expiresAt?: string;
  contentType?: string;
  size?: number;
  displayOrder?: number;
  fileName?: string;
};

/**
 * ✅ ListDetail DTO（型ガイド用）
 */
export type ListDetailDTO = ListDTO & {
  createdByName?: string;
  updatedByName?: string;

  createdBy?: string;
  updatedBy?: string;

  createdAt?: string;
  updatedAt?: string;

  // ✅ listImage bucket からの画像URL
  imageId?: string;
  imageUrls?: string[];
};

/**
 * ===========
 * Internal helpers
 * ===========
 */

function s(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * ✅ list のドキュメントID用の正規化
 * - これは "listId__imageId" など事故混入の保険
 * - ただし inventoryId (pb__tb) には絶対に使わない（方針A）
 */
function normalizeListDocId(v: unknown): string {
  const id = s(v);
  if (!id) return "";
  return id.split("__")[0];
}

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = Number(v);
  if (!Number.isFinite(n)) return null;
  return n;
}

/**
 * ✅ objectPath を URL パスとして安全にする
 * - "/" はパス区切りとして残したいのでセグメント単位で encodeURIComponent
 * - 例: "lists/xxx/スクショ (1).png" を安全なURLへ
 */
function encodeGcsObjectPath(objectPath: string): string {
  const raw = String(objectPath ?? "").trim().replace(/^\/+/, "");
  if (!raw) return "";
  return raw
    .split("/")
    .map((seg) => encodeURIComponent(seg))
    .join("/");
}

function buildPublicGcsUrl(bucket: string, objectPath: string): string {
  const b = s(bucket) || LIST_IMAGE_BUCKET;
  const opRaw = String(objectPath ?? "").trim().replace(/^\/+/, "");
  const op = encodeGcsObjectPath(opRaw);
  if (!b || !op) return "";
  return `https://storage.googleapis.com/${b}/${op}`;
}

/**
 * ✅ ListImage から "表示用URL" を解決
 * 優先順位:
 * 1) publicUrl/url/signedUrl 等（= backend が完成URLを返す場合）
 * 2) bucket + objectPath から public URL を組み立て（objectPathはURLエンコード）
 */
function resolveListImageUrl(img: ListImageDTO): string {
  const u =
    s((img as any)?.publicUrl) ||
    s((img as any)?.publicURL) ||
    s((img as any)?.url) ||
    s((img as any)?.URL) ||
    s((img as any)?.signedUrl) ||
    s((img as any)?.signedURL) ||
    s((img as any)?.uploadUrl) ||
    s((img as any)?.uploadURL);

  // ✅ ここは backend が返したURLを尊重（既にエンコード済み/署名付き等の可能性）
  if (u) return u;

  const bucket = s((img as any)?.bucket) || s((img as any)?.Bucket) || "";
  const objectPath =
    s((img as any)?.objectPath) ||
    s((img as any)?.ObjectPath) ||
    s((img as any)?.path) ||
    s((img as any)?.Path) ||
    "";

  const built = buildPublicGcsUrl(bucket, objectPath);
  return built;
}

function asNumber(v: unknown, fallback = 0): number {
  const n = Number(v);
  if (!Number.isFinite(n)) return fallback;
  return n;
}

function parseDateMs(v: unknown): number {
  const t = s(v);
  if (!t) return 0;
  const ms = Date.parse(t);
  if (!Number.isFinite(ms)) return 0;
  return ms;
}

function normalizeListImageUrls(
  listImages: ListImageDTO[],
  primaryImageId?: string,
): string[] {
  const pid = s(primaryImageId);

  const rows = (Array.isArray(listImages) ? listImages : [])
    .map((img) => {
      const id = s((img as any)?.id) || s((img as any)?.ID) || s((img as any)?.imageId);
      const url = resolveListImageUrl(img);
      const displayOrder =
        asNumber((img as any)?.displayOrder, 0) ||
        asNumber((img as any)?.DisplayOrder, 0) ||
        0;

      const createdAtMs =
        parseDateMs((img as any)?.createdAt) ||
        parseDateMs((img as any)?.CreatedAt) ||
        0;

      return { id, url, displayOrder, createdAtMs };
    })
    .filter((x) => Boolean(x.url));

  rows.sort((a, b) => {
    if (a.displayOrder !== b.displayOrder) return a.displayOrder - b.displayOrder;
    if (a.createdAtMs !== b.createdAtMs) return a.createdAtMs - b.createdAtMs;
    return a.id.localeCompare(b.id);
  });

  const out: string[] = [];
  const seen = new Set<string>();
  let primaryUrl = "";

  for (const r of rows) {
    const url = s(r.url);
    if (!url || seen.has(url)) continue;
    seen.add(url);

    if (pid && s(r.id) === pid && !primaryUrl) {
      primaryUrl = url;
      continue;
    }
    out.push(url);
  }

  if (primaryUrl) return [primaryUrl, ...out];
  return out;
}

/**
 * ✅ ListDetailDTO の imageUrls を保証する
 * - backend が imageUrls を返していればそれを優先
 * - 空なら /lists/{id}/images から組み立てて補完
 */
async function ensureDetailHasImageUrls(dto: ListDTO, listIdRaw: string): Promise<ListDTO> {
  const listId = normalizeListDocId(listIdRaw);
  const anyDto = dto as any;

  const currentUrls = Array.isArray(anyDto?.imageUrls) ? (anyDto.imageUrls as any[]) : [];
  const normalizedCurrent = currentUrls.map((x) => s(x)).filter(Boolean);

  if (normalizedCurrent.length > 0) {
    return {
      ...dto,
      imageUrls: normalizedCurrent,
    };
  }

  // fallback: images endpoint から生成
  try {
    const imgs = await fetchListImagesHTTP(listId);
    const urls = normalizeListImageUrls(imgs, s(anyDto?.imageId));

    if (urls.length === 0) return dto;

    return {
      ...dto,
      imageUrls: urls,
    };
  } catch {
    // 画像取得に失敗しても detail 自体は返す（画面を壊さない）
    return dto;
  }
}

/**
 * ✅ create 用の prices を正規化する（modelId + price ONLY）
 *
 * - modelId が無い行があれば例外（送信しない）
 * - price が null / NaN なら例外（Go 側が非nullableの可能性が高い）
 */
function normalizePricesForBackend(
  rows: CreateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r) => {
    const modelId = s((r as any)?.modelId);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      throw new Error("missing_modelId_in_priceRows");
    }

    if (priceMaybe === null) {
      throw new Error("missing_price_in_priceRows");
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ update 用: modelId を row.modelId または row.id から取得する
 */
function normalizePricesForBackendUpdate(
  rows: UpdateListInput["priceRows"],
): Array<{ modelId: string; price: number }> {
  if (!Array.isArray(rows)) return [];

  return rows.map((r, idx) => {
    const modelId = s((r as any)?.modelId) || s((r as any)?.id);
    const priceMaybe = toNumberOrNull((r as any)?.price);

    if (!modelId) {
      throw new Error(`missing_modelId_in_priceRows_at_${idx}`);
    }
    if (priceMaybe === null) {
      throw new Error(`missing_price_in_priceRows_at_${idx}`);
    }

    return { modelId, price: priceMaybe };
  });
}

/**
 * ✅ CreateList payload（最小）
 * - 「create時に送るのは modelId と price」の方針を厳守
 * - ✅ 方針A: inventoryId は pb__tb をそのまま送る
 */
function buildCreateListPayloadArray(input: CreateListInput): Record<string, any> {
  const u = auth.currentUser;
  const uid = s(u?.uid);

  const inventoryId = s(input?.inventoryId); // ✅ splitしない
  const id = normalizeListDocId(input?.id) || inventoryId; // ✅ id未指定なら inventoryId を採用（従来方針）

  if (!id) {
    throw new Error("missing_id");
  }

  const title = s(input?.title);
  if (!title) {
    throw new Error("missing_title");
  }

  const prices = normalizePricesForBackend(input?.priceRows);

  return {
    id,
    inventoryId,
    title,
    description: String(input?.description ?? ""),
    assigneeId: s(input?.assigneeId) || undefined,
    createdBy: s(input?.createdBy) || uid || "system",
    prices, // Array<{modelId, price}>
  };
}

/**
 * ✅ NEW: Update payload（最小）
 */
function buildUpdateListPayloadArray(input: UpdateListInput): Record<string, any> {
  const u = auth.currentUser;
  const uid = s(u?.uid);

  const title = s(input?.title);
  const description =
    input?.description === undefined ? undefined : String(input?.description ?? "");

  const prices = normalizePricesForBackendUpdate(input?.priceRows);

  // decision -> status
  let status: string | undefined = undefined;
  if (input?.decision === "list") status = "listing";
  if (input?.decision === "hold") status = "hold";

  const payload: Record<string, any> = {
    title: title || undefined,
    description,
    assigneeId: s(input?.assigneeId) || undefined,
    prices,
    status,
    decision: undefined,
    updatedBy: s(input?.updatedBy) || uid || undefined,
  };

  for (const k of Object.keys(payload)) {
    if (payload[k] === undefined) delete payload[k];
  }

  return payload;
}

/**
 * ✅ fallback: prices を map で送る版
 */
function buildCreateListPayloadMap(input: CreateListInput): Record<string, any> {
  const base = buildCreateListPayloadArray(input);
  const pricesArray = Array.isArray((base as any).prices)
    ? ((base as any).prices as any[])
    : [];

  const pricesMap: Record<string, number> = {};
  for (const p of pricesArray) {
    const modelId = s((p as any)?.modelId);
    const price = Number((p as any)?.price);
    if (!modelId || !Number.isFinite(price)) continue;
    pricesMap[modelId] = price;
  }

  return {
    ...base,
    prices: pricesMap,
  };
}

/**
 * ✅ NEW: update fallback: prices を map で送る版
 */
function buildUpdateListPayloadMap(input: UpdateListInput): Record<string, any> {
  const base = buildUpdateListPayloadArray(input);
  const pricesArray = Array.isArray((base as any).prices)
    ? ((base as any).prices as any[])
    : [];

  const pricesMap: Record<string, number> = {};
  for (const p of pricesArray) {
    const modelId = s((p as any)?.modelId);
    const price = Number((p as any)?.price);
    if (!modelId || !Number.isFinite(price)) continue;
    pricesMap[modelId] = price;
  }

  return {
    ...base,
    prices: pricesMap,
  };
}

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) throw new Error("not_authenticated");
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;

  // ✅ debug log
  debug?: {
    tag: string;
    url: string;
    method: string;
    body?: unknown;
  };
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  if (args.debug) {
    try {
      const bodyStr =
        args.debug.body === undefined ? undefined : JSON.stringify(args.debug.body);
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag}`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        bodyJSON: bodyStr,
      });
    } catch (e) {
      console.log(`[list/listRepositoryHTTP] ${args.debug.tag} (stringify_failed)`, {
        method: args.debug.method,
        url: args.debug.url,
        body: args.debug.body,
        err: String(e),
      });
    }
  }

  let bodyText: string | undefined = undefined;
  if (args.body !== undefined) {
    try {
      bodyText = JSON.stringify(args.body);
    } catch {
      throw new Error("invalid_json_stringify");
    }
  }

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

    console.log(`[list/listRepositoryHTTP] response error`, {
      method: args.method,
      url,
      status: res.status,
      raw: text,
      json,
    });

    throw new Error(String(msg));
  }

  return json as T;
}

function extractItemsArrayFromAny(json: any): any[] {
  if (Array.isArray(json)) return json;
  if (json && typeof json === "object") {
    if (Array.isArray((json as any).items)) return (json as any).items;
    if (Array.isArray((json as any).Items)) return (json as any).Items;
    if (Array.isArray((json as any).data)) return (json as any).data;
  }
  return [];
}

function extractFirstItemFromAny(json: any): any | null {
  if (!json) return null;
  if (Array.isArray(json)) return json[0] ?? null;

  if (json && typeof json === "object") {
    if ((json as any).id) return json;

    const items = extractItemsArrayFromAny(json);
    return items[0] ?? null;
  }

  return null;
}

/**
 * ✅ Signed URL レスポンスを “呼び出し側が使える形” に正規化する
 * - backend: uploadUrl/publicUrl/objectPath/id...
 * - legacy: signedUrl/publicUrl/objectPath...
 */
function normalizeSignedListImageUploadDTO(raw: any): SignedListImageUploadDTO {
  const id = s(raw?.id) || s(raw?.ID) || undefined;
  const bucket = s(raw?.bucket) || s(raw?.Bucket) || undefined;

  const objectPath =
    s(raw?.objectPath) ||
    s(raw?.ObjectPath) ||
    s(raw?.path) ||
    s(raw?.Path) ||
    s(raw?.id) || // backend では id=objectPath のことがある
    "";

  const signedUrl =
    s(raw?.signedUrl) ||
    s(raw?.signedURL) ||
    s(raw?.uploadUrl) ||
    s(raw?.uploadURL) ||
    "";

  const publicUrl =
    s(raw?.publicUrl) || s(raw?.publicURL) || s(raw?.url) || s(raw?.URL) || "";

  // もし publicUrl が無いなら bucket+objectPath から組み立て（表示用）
  const builtPublicUrl = publicUrl || buildPublicGcsUrl(bucket || "", objectPath);

  const expiresAt = s(raw?.expiresAt) || s(raw?.ExpiresAt) || undefined;
  const contentType = s(raw?.contentType) || s(raw?.ContentType) || undefined;

  const size = Number(raw?.size);
  const displayOrder = Number(raw?.displayOrder);
  const fileName = s(raw?.fileName) || s(raw?.FileName) || undefined;

  if (!objectPath || !signedUrl) {
    // inventory 側のエラーハンドリングが msg === "signed_url_response_invalid" を見てるので合わせる
    throw new Error("signed_url_response_invalid");
  }

  return {
    id,
    bucket,
    objectPath,
    signedUrl,
    publicUrl: builtPublicUrl || undefined,
    expiresAt,
    contentType,
    size: Number.isFinite(size) ? size : undefined,
    displayOrder: Number.isFinite(displayOrder) ? displayOrder : undefined,
    fileName,
  };
}

/**
 * ===========
 * API
 * ===========
 */

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  const payloadArray = buildCreateListPayloadArray(input);

  console.log("[list/listRepositoryHTTP] createListHTTP payload", payloadArray);

  try {
    return await requestJSON<ListDTO>({
      method: "POST",
      path: "/lists",
      body: payloadArray,
      debug: {
        tag: "POST /lists",
        url: `${API_BASE}/lists`,
        method: "POST",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildCreateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] createListHTTP retry payload(map)", payloadMap);

      return await requestJSON<ListDTO>({
        method: "POST",
        path: "/lists",
        body: payloadMap,
        debug: {
          tag: "POST /lists (retry map)",
          url: `${API_BASE}/lists`,
          method: "POST",
          body: payloadMap,
        },
      });
    }

    throw e;
  }
}

/**
 * ✅ Update list
 * PUT /lists/{id}
 */
export async function updateListByIdHTTP(input: UpdateListInput): Promise<ListDTO> {
  const listId = normalizeListDocId(input?.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payloadArray = buildUpdateListPayloadArray(input);

  console.log("[list/listRepositoryHTTP] updateListByIdHTTP payload", {
    listId,
    payload: payloadArray,
  });

  try {
    return await requestJSON<ListDTO>({
      method: "PUT",
      path: `/lists/${encodeURIComponent(listId)}`,
      body: payloadArray,
      debug: {
        tag: `PUT /lists/${listId}`,
        url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
        method: "PUT",
        body: payloadArray,
      },
    });
  } catch (e) {
    const msg = String(e instanceof Error ? e.message : e);

    if (msg === "invalid json") {
      const payloadMap = buildUpdateListPayloadMap(input);

      console.log("[list/listRepositoryHTTP] updateListByIdHTTP retry payload(map)", {
        listId,
        payload: payloadMap,
      });

      return await requestJSON<ListDTO>({
        method: "PUT",
        path: `/lists/${encodeURIComponent(listId)}`,
        body: payloadMap,
        debug: {
          tag: `PUT /lists/${listId} (retry map)`,
          url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
          method: "PUT",
          body: payloadMap,
        },
      });
    }

    throw e;
  }
}

/**
 * ✅ List lists
 * GET /lists
 */
export async function fetchListsHTTP(): Promise<ListDTO[]> {
  const json = await requestJSON<any>({
    method: "GET",
    path: "/lists",
  });

  const items = extractItemsArrayFromAny(json);
  return items as ListDTO[];
}

/**
 * GET /lists/{id}
 */
export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = normalizeListDocId(listId);
  if (!id) {
    throw new Error("invalid_list_id");
  }

  const dto0 = await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
    debug: {
      tag: `GET /lists/${id}`,
      url: `${API_BASE}/lists/${encodeURIComponent(id)}`,
      method: "GET",
    },
  });

  const dto = await ensureDetailHasImageUrls(dto0, id);

  try {
    const anyDto = dto as any;
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok", {
      listId: id,
      hasCreatedByName: Boolean(s(anyDto?.createdByName)),
      createdBy: s(anyDto?.createdBy),
      createdByName: s(anyDto?.createdByName),
      updatedBy: s(anyDto?.updatedBy),
      updatedByName: s(anyDto?.updatedByName),
      createdAt: s(anyDto?.createdAt),
      updatedAt: s(anyDto?.updatedAt),
      imageId: s(anyDto?.imageId),
      imageUrlsCount: Array.isArray(anyDto?.imageUrls) ? anyDto.imageUrls.length : 0,
      keys: anyDto && typeof anyDto === "object" ? Object.keys(anyDto) : [],
      dto,
    });
  } catch (e) {
    console.log("[list/listRepositoryHTTP] fetchListByIdHTTP ok (log_failed)", {
      listId: id,
      err: String(e),
    });
  }

  return dto;
}

/**
 * ✅ ListDetail 用
 */
export async function fetchListDetailHTTP(args: {
  listId: string;
  inventoryIdHint?: string;
}): Promise<ListDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) {
    throw new Error("invalid_list_id");
  }

  console.log("[list/listRepositoryHTTP] fetchListDetailHTTP start", {
    listId,
    inventoryIdHint: s(args.inventoryIdHint),
    url: `${API_BASE}/lists/${encodeURIComponent(listId)}`,
  });

  try {
    const dto = await fetchListByIdHTTP(listId);

    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
      source: "GET /lists/{id}",
      listId,
      createdByName: s((dto as any)?.createdByName),
      updatedByName: s((dto as any)?.updatedByName),
      imageUrlsCount: Array.isArray((dto as any)?.imageUrls)
        ? (dto as any).imageUrls.length
        : 0,
      dto,
    });

    return dto;
  } catch (e1) {
    // ✅ inventoryIdHint は pb__tb をそのまま使う（splitしない）
    const inv = s(args.inventoryIdHint) || listId;

    console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback start", {
      listId,
      inventoryId: inv,
      url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
      err: String(e1 instanceof Error ? e1.message : e1),
    });

    try {
      const json = await requestJSON<any>({
        method: "GET",
        path: `/lists?inventoryId=${encodeURIComponent(inv)}`,
        debug: {
          tag: `GET /lists?inventoryId=${inv}`,
          url: `${API_BASE}/lists?inventoryId=${encodeURIComponent(inv)}`,
          method: "GET",
        },
      });

      const first0 = extractFirstItemFromAny(json);
      if (!first0) throw new Error("not_found");

      const first = await ensureDetailHasImageUrls(first0 as ListDTO, listId);

      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP resolved", {
        source: "GET /lists?inventoryId=xxx",
        listId,
        inventoryId: inv,
        createdByName: s((first as any)?.createdByName),
        updatedByName: s((first as any)?.updatedByName),
        imageUrlsCount: Array.isArray((first as any)?.imageUrls)
          ? (first as any).imageUrls.length
          : 0,
        dto: first,
        raw: json,
      });

      return first as ListDTO;
    } catch (e2) {
      console.log("[list/listRepositoryHTTP] fetchListDetailHTTP fallback failed", {
        listId,
        inventoryId: inv,
        err: String(e2 instanceof Error ? e2.message : e2),
      });
      throw e1;
    }
  }
}

/**
 * GET /lists/{id}/aggregate
 */
export async function fetchListAggregateHTTP(listId: string): Promise<ListAggregateDTO> {
  const id = normalizeListDocId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListAggregateDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/aggregate`,
  });
}

/**
 * GET /lists/{id}/images
 */
export async function fetchListImagesHTTP(listId: string): Promise<ListImageDTO[]> {
  const id = normalizeListDocId(listId);
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
}

/**
 * ✅ NEW: listImage bucket の「表示用URL配列」を取得
 */
export async function fetchListImageUrlsHTTP(args: {
  listId: string;
  primaryImageId?: string;
}): Promise<string[]> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const imgs = await fetchListImagesHTTP(listId);
  return normalizeListImageUrls(imgs, args.primaryImageId);
}

/**
 * ✅ NEW: signed-url 発行（Policy A）
 * POST /lists/{id}/images/signed-url
 */
export async function issueListImageSignedUrlHTTP(args: {
  listId: string;
  fileName: string;
  contentType: string;
  size: number;
  displayOrder: number;
}): Promise<SignedListImageUploadDTO> {
  // ✅ ここは list の docId なので normalize してOK（事故混入対策）
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    fileName: s(args.fileName),
    contentType: s(args.contentType) || "application/octet-stream",
    size: Number(args.size || 0),
    displayOrder: Number(args.displayOrder || 0),
  };

  // backend の返却キー揺れ（uploadUrl / signedUrl / publicUrl など）をここで吸収する
  const raw = await requestJSON<any>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images/signed-url`,
    body: payload,
    debug: {
      tag: `POST /lists/${listId}/images/signed-url`,
      url: `${API_BASE}/lists/${encodeURIComponent(listId)}/images/signed-url`,
      method: "POST",
      body: payload,
    },
  });

  return normalizeSignedListImageUploadDTO(raw);
}

/**
 * POST /lists/{id}/images
 * - GCS objectPath を登録する（アップロード自体は別途）
 */
export async function saveListImageFromGCSHTTP(args: {
  listId: string;
  id: string; // ListImage.ID
  fileName?: string;
  bucket?: string; // optional
  objectPath: string;
  size: number; // bytes
  displayOrder: number;
  createdBy?: string;
  createdAt?: string; // RFC3339 optional
}): Promise<ListImageDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    id: String(args.id ?? "").trim(),
    fileName: String(args.fileName ?? "").trim(),
    bucket: String(args.bucket ?? "").trim(),
    objectPath: String(args.objectPath ?? "").trim(),
    size: Number(args.size ?? 0),
    displayOrder: Number(args.displayOrder ?? 0),
    createdBy: String(args.createdBy ?? "").trim(),
    createdAt: args.createdAt ? String(args.createdAt).trim() : undefined,
  };

  return await requestJSON<ListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: payload,
  });
}

/**
 * PUT /lists/{id}/primary-image
 */
export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string; // RFC3339 optional
}): Promise<ListDTO> {
  const listId = normalizeListDocId(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: String(args.imageId ?? "").trim(),
    updatedBy: args.updatedBy ? String(args.updatedBy).trim() : undefined,
    now: args.now ? String(args.now).trim() : undefined,
  };

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}
