// frontend/console/inventory/src/application/listCreateService.tsx

import type { RefObject } from "react";

import type { PriceRow } from "../../../list/src/presentation/hook/usePriceCard";

import {
  fetchListCreateDTO,
  type ListCreateDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ✅ Firebase Auth（IDトークン / uid 取得）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ list create (POST /lists) + listImage APIs
import {
  API_BASE,
  createListHTTP,
  saveListImageFromGCSHTTP,
  setListPrimaryImageHTTP,
  LIST_IMAGE_BUCKET,
  type CreateListInput,
  type ListDTO,
  type ListImageDTO,
} from "../../../list/src/infrastructure/http/listRepositoryHTTP";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ✅ Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
export type ImageInputRef = RefObject<HTMLInputElement | null>;

export type ListCreateRouteParams = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

export type ResolvedListCreateParams = {
  inventoryId: string;
  productBlueprintId: string;
  tokenBlueprintId: string;
  raw: ListCreateRouteParams;
};

export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  return {
    inventoryId: s(raw?.inventoryId),
    productBlueprintId: s(raw?.productBlueprintId),
    tokenBlueprintId: s(raw?.tokenBlueprintId),
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  return (
    Boolean(p.inventoryId) ||
    (Boolean(p.productBlueprintId) && Boolean(p.tokenBlueprintId))
  );
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  return {
    inventoryId: p.inventoryId || undefined,
    productBlueprintId: p.productBlueprintId || undefined,
    tokenBlueprintId: p.tokenBlueprintId || undefined,
  };
}

export function getInventoryIdFromDTO(
  dto: ListCreateDTO | null | undefined,
): string {
  return s((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
}

export function shouldRedirectToInventoryIdRoute(args: {
  currentInventoryId: string;
  gotInventoryId: string;
  alreadyRedirected: boolean;
}): boolean {
  return (
    !args.alreadyRedirected &&
    !args.currentInventoryId &&
    Boolean(args.gotInventoryId)
  );
}

export function buildInventoryDetailPath(pbId: string, tbId: string): string {
  const pb = s(pbId);
  const tb = s(tbId);
  if (!pb || !tb) return "/inventory";
  return `/inventory/detail/${encodeURIComponent(pb)}/${encodeURIComponent(tb)}`;
}

export function buildInventoryListCreatePath(inventoryId: string): string {
  const id = s(inventoryId);
  if (!id) return "/inventory/list/create";
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  // ✅ 詳細へは pb/tb で戻す
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
  // ✅ 作成後も pb/tb があれば detail へ
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  return "/inventory";
}

export function extractDisplayStrings(dto: ListCreateDTO | null): {
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  return {
    productBrandName: s(dto?.productBrandName),
    productName: s(dto?.productName),
    tokenBrandName: s(dto?.tokenBrandName),
    tokenName: s(dto?.tokenName),
  };
}

/**
 * ✅ backend の ListCreateDTO.priceRows を PriceCard 用 PriceRow[] に変換
 * - dto 側に priceRows が無ければ []
 *
 * ※ ここは UI 用のため、size/color/stock/rgb を残してOK
 *    （POST /lists に送る形は buildCreateListInput で「modelId+price のみ」に射影する）
 */
export function mapDTOToPriceRows(dto: ListCreateDTO | null): PriceRow[] {
  const rowsAny: any[] = Array.isArray((dto as any)?.priceRows)
    ? ((dto as any).priceRows as any[])
    : Array.isArray((dto as any)?.PriceRows)
      ? ((dto as any).PriceRows as any[])
      : [];

  return rowsAny.flatMap((r: any) => {
    const size = s(r?.size ?? r?.Size) || "-";
    const color = s(r?.color ?? r?.Color) || "-";
    const stock = Number(r?.stock ?? r?.Stock ?? 0);
    const rgb = r?.rgb ?? r?.RGB; // number|string|null|undefined 想定
    const price = r?.price ?? r?.Price;

    const safeStock = Number.isFinite(stock) ? stock : 0;

    const row: PriceRow = {
      size,
      color,
      stock: safeStock,
      rgb: rgb as any,
      price: price === undefined ? null : (price as any),
    };

    return [row];
  });
}

/**
 * ✅ ListCreateDTO を取得する（Hook からはこれだけ呼ぶ）
 */
export async function loadListCreateDTOFromParams(
  p: ResolvedListCreateParams,
): Promise<ListCreateDTO> {
  const input = buildListCreateFetchInput(p);
  return await fetchListCreateDTO(input);
}

// ============================================================
// ✅ PriceRows: DTO -> (PriceRow + modelId)
// ============================================================

/**
 * ✅ PriceRow に modelId を保持させる（POST /lists で必須）
 * - usePriceCard は余分なフィールドがあっても問題ないので、そのまま渡せる
 */
export type PriceRowEx = PriceRow & {
  modelId: string; // ✅ 必須
};

/**
 * ✅ DTO の priceRows から modelId を埋める
 * - マッチは (size,color) を基本にし、最後に index fallback（DTO順が一致する場合）を使う
 */
export function attachModelIdsFromDTO(dto: any, baseRows: PriceRow[]): PriceRowEx[] {
  const dtoRows: any[] = Array.isArray(dto?.priceRows) ? dto.priceRows : [];

  const keyToModelId = new Map<string, string>();
  for (const dr of dtoRows) {
    const size = s(dr?.size);
    const color = s(dr?.color);
    const modelId = s(dr?.modelId);
    if (!size || !color || !modelId) continue;
    keyToModelId.set(`${size}__${color}`, modelId);
  }

  return baseRows.map((r, idx) => {
    const size = s((r as any)?.size);
    const color = s((r as any)?.color);
    const byKey = keyToModelId.get(`${size}__${color}`) ?? "";
    const byIndex = s(dtoRows[idx]?.modelId);
    const modelId = byKey || byIndex;

    return {
      ...(r as any),
      modelId,
    } as PriceRowEx;
  });
}

/**
 * ✅ Hook 側の初期化用（DTO -> PriceRowEx[]）
 */
export function initPriceRowsFromDTO(dto: ListCreateDTO | null): PriceRowEx[] {
  const base = mapDTOToPriceRows(dto);
  return attachModelIdsFromDTO(dto as any, base);
}

// ============================================================
// ✅ POST /lists: 期待値どおり「modelId + price のみ」
// ============================================================

export type CreateListPriceRow = {
  modelId: string;
  price: number | null;
};

function toNumberOrNull(v: unknown): number | null {
  if (v === null || v === undefined) return null;
  const n = typeof v === "number" ? v : Number(String(v).trim());
  if (!Number.isFinite(n)) return null;
  return Math.floor(n);
}

/**
 * ✅ Hook から渡された priceRows を「modelId+price」に正規化する
 * - Hook 側が PriceRowEx を渡してくる想定（modelId を含む）
 * - 互換のため ModelID なども拾う
 * - size/color/stock/rgb 等は POST には一切含めない
 */
export function normalizeCreateListPriceRows(rows: any[]): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];
  return arr.map((r) => {
    const modelId = s((r as any)?.modelId ?? (r as any)?.ModelID);
    const price = toNumberOrNull((r as any)?.price);
    return { modelId, price };
  });
}

/**
 * ✅ POST /lists 用の payload を組み立てる
 * - 期待値：listRepositoryHTTP.tsx へは {modelId, price} のみ渡す
 */
export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  // ✅ Hook 側が PriceRowEx（modelId含む）を渡してくるので any[] で受ける
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    inventoryId: s(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    // ✅ ここが重要：modelId と price 以外は送らない
    priceRows: priceRows as any,
  } as CreateListInput;
}

/**
 * ✅ 入力バリデーション（UI 側の要件）
 * - title が空欄 → エラー
 * - modelId が欠ける行がある → エラー
 * - price が 0（または未入力/0のみ） → エラー
 */
export function validateCreateListInput(input: CreateListInput): void {
  const title = s((input as any)?.title);
  if (!title) {
    throw new Error("タイトルを入力してください。");
  }

  const rows = Array.isArray((input as any)?.priceRows) ? (input as any).priceRows : [];
  if (rows.length === 0) {
    throw new Error("価格が未設定です（価格行がありません）。");
  }

  const missingModelId = rows.find((r: any) => !s(r?.modelId ?? r?.ModelID));
  if (missingModelId) {
    throw new Error("価格行に modelId が含まれていません。");
  }

  // 価格が1つも入っていない or 0 しか無い場合は NG
  const hasPositivePrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n > 0;
  });
  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

  // 念のため「0円」の行が混ざっていたらエラー
  const hasZeroPrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n === 0;
  });
  if (hasZeroPrice) {
    throw new Error("価格に 0 円が含まれています。0 円は指定できません。");
  }
}

// ============================================================
// ✅ ListImage: (Policy A) signed-url -> PUT -> metadata -> primary
// ============================================================

export function dedupeFiles(prev: File[], add: File[]): File[] {
  const exists = new Set(prev.map((f) => `${f.name}__${f.size}__${f.lastModified}`));
  const filtered = add.filter((f) => !exists.has(`${f.name}__${f.size}__${f.lastModified}`));
  return [...prev, ...filtered];
}

function randomHex(bytes = 8): string {
  try {
    const a = new Uint8Array(Math.max(1, bytes));
    crypto.getRandomValues(a);
    return Array.from(a)
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");
  } catch {
    return String(Date.now());
  }
}

function getListIdFromListDTO(dto: ListDTO, fallback = ""): string {
  return (
    s((dto as any)?.id) ||
    s((dto as any)?.ID) ||
    s((dto as any)?.listId) ||
    s((dto as any)?.ListID) ||
    s(fallback)
  );
}

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) throw new Error("not_authenticated");
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "POST" | "GET" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: any;
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: args.body === undefined ? undefined : JSON.stringify(args.body),
  });

  const text = await res.text().catch(() => "");
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

async function putFileToSignedUrl(args: { signedUrl: string; file: File }): Promise<void> {
  const url = s(args.signedUrl);
  const file = args.file;
  if (!url) throw new Error("missing_signed_url");

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": file.type || "application/octet-stream",
    },
    body: file,
  });

  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(`listImage_upload_failed_${res.status}_${t || "no_body"}`);
  }
}

type SignedListImageUpload = {
  bucket?: string;
  objectPath: string;
  signedUrl: string;
  publicUrl?: string;
};

/**
 * ✅ 署名付きURL発行（Policy A）
 * - backend に追加される想定の API
 * - 期待: { bucket?, objectPath, signedUrl, publicUrl? }
 */
async function issueListImageSignedUrl(args: {
  listId: string;
  fileName: string;
  contentType: string;
  size: number;
  displayOrder: number;
}): Promise<SignedListImageUpload> {
  const listId = s(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  return await requestJSON<SignedListImageUpload>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images/signed-url`,
    body: {
      fileName: s(args.fileName),
      contentType: s(args.contentType) || "application/octet-stream",
      size: Number(args.size || 0),
      displayOrder: Number(args.displayOrder || 0),
    },
  });
}

/**
 * ✅ 複数画像を Policy A（signed-url）でアップロード→メタ登録→primary 設定
 */
export async function uploadListImagesPolicyA(args: {
  listId: string;
  files: File[];
  mainImageIndex: number;
  createdBy?: string;
}): Promise<{ registered: Array<{ imageId: string; displayOrder: number }>; primaryImageId?: string }> {
  const listId = s(args.listId);
  const files = Array.isArray(args.files) ? args.files : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex)) ? Number(args.mainImageIndex) : 0;

  if (!listId) throw new Error("invalid_list_id");
  if (files.length === 0) return { registered: [] };

  if (!files[mainImageIndex]) {
    throw new Error("メイン画像が選択されていません。");
  }

  const uid = s(args.createdBy) || s(auth.currentUser?.uid) || "system";
  const now = new Date().toISOString();

  const registered: Array<{ imageId: string; displayOrder: number }> = [];

  for (let i = 0; i < files.length; i++) {
    const file = files[i];
    if (!file) continue;

    const signed = await issueListImageSignedUrl({
      listId,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      size: file.size || 0,
      displayOrder: i,
    });

    const bucket = s(signed.bucket) || LIST_IMAGE_BUCKET || "listimage";
    const objectPath = s(signed.objectPath);
    const signedUrl = s(signed.signedUrl);

    if (!objectPath || !signedUrl) {
      throw new Error("signed_url_response_invalid");
    }

    await putFileToSignedUrl({ signedUrl, file });

    const imageId = randomHex(12);

    await saveListImageFromGCSHTTP({
      listId,
      id: imageId,
      fileName: s(file.name),
      bucket,
      objectPath,
      size: Number(file.size || 0),
      displayOrder: i,
      createdBy: uid,
      createdAt: now,
    });

    registered.push({ imageId, displayOrder: i });
  }

  const primary =
    registered.find((x) => x.displayOrder === mainImageIndex) || registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,
      imageId: primary.imageId,
      updatedBy: uid,
      now, // optional (repository が受け取れる前提)
    } as any);
  }

  return { registered, primaryImageId: primary?.imageId };
}

// ============================================================
// ✅ list 作成（POST /lists） + 画像（Policy A）もここで完結
// ============================================================

/**
 * ✅ list 作成（POST /lists）
 * ✅ その後、必要なら listImages を Policy A（signed-url）で登録して primary も設定する
 */
export async function createListWithImages(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[]; // PriceRowEx[] 想定
  decision: "list" | "hold";
  assigneeId?: string;

  // images (optional)
  images?: File[];
  mainImageIndex?: number;
}): Promise<ListDTO> {
  const images = Array.isArray(args.images) ? args.images : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  // 1) build + validate
  const input = buildCreateListInput({
    params: args.params,
    listingTitle: args.listingTitle,
    description: args.description,
    priceRows: args.priceRows,
    decision: args.decision,
    assigneeId: args.assigneeId,
  });

  validateCreateListInput(input);

  // 画像がある場合は main が存在すること（空配列ならOK）
  if (images.length > 0 && !images[mainImageIndex]) {
    throw new Error("メイン画像が選択されていません。");
  }

  // 2) create list
  const created = await createListHTTP(input);

  const listId = getListIdFromListDTO(
    created,
    s((input as any)?.id) || s((input as any)?.inventoryId),
  );
  if (!listId) {
    throw new Error("created_list_missing_id");
  }

  // 3) images (Policy A)
  if (images.length > 0) {
    const r = await uploadListImagesPolicyA({
      listId,
      files: images,
      mainImageIndex,
      createdBy: s(auth.currentUser?.uid) || undefined,
    });

    // primary 設定 API が ListDTO を返す仕様なら、それを返したいが、
    // uploadListImagesPolicyA はここでは返せないため created を返す（画面は遷移する想定）
    // 必要なら setListPrimaryImageHTTP の戻りを使う実装に切り替え可能
    void r;
  }

  return created;
}

// ============================================================
// ✅ 互換: 単一画像 POST (/lists/{id}/images) + primary-image
// （既存呼び出しがある場合に備えて残す）
// ============================================================

export type PostListImageArgs = {
  /**
   * ① 既に GCS にある画像を登録したい場合:
   * - objectPath を渡す（bucket は省略可）
   */
  objectPath?: string;
  bucket?: string;

  /**
   * ② 画面の file input から登録したい場合:
   * - fileInputRef を渡す
   * - objectPath が未指定なら、この service 側で生成する
   *
   * NOTE:
   * - 「アップロード」まで必要な場合は uploadUrl（署名付きURL）を渡してください。
   * - uploadUrl が無い場合は public GCS URL に PUT を試みます（403 なら失敗します）。
   */
  fileInputRef?: ImageInputRef;
  uploadUrl?: string; // signed URL (PUT)

  // optional metadata
  fileName?: string;
  displayOrder?: number;
  createdBy?: string;
  createdAt?: string; // RFC3339 optional

  // ✅ 登録した画像を primary にする（create後のデフォルト想定）
  setPrimary?: boolean;
};

function takeFirstFile(ref?: ImageInputRef): File | null {
  try {
    const el = ref?.current;
    const f = el?.files && el.files.length > 0 ? el.files[0] : null;
    return f ?? null;
  } catch {
    return null;
  }
}

function encodeGcsPath(objectPath: string): string {
  const p = String(objectPath ?? "").trim().replace(/^\/+/, "");
  // segment ごとに encode して / を維持
  return p
    .split("/")
    .map((seg) => encodeURIComponent(seg))
    .join("/");
}

async function putFileToUrl(uploadUrl: string, file: File): Promise<void> {
  const url = String(uploadUrl ?? "").trim();
  if (!url) throw new Error("missing_upload_url");

  const res = await fetch(url, {
    method: "PUT",
    headers: {
      "Content-Type": file.type || "application/octet-stream",
    },
    body: file,
  });

  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(`upload_failed_${res.status}_${t || "no_body"}`);
  }
}

async function putFileToPublicGcs(
  bucket: string,
  objectPath: string,
  file: File,
): Promise<void> {
  const b = s(bucket) || LIST_IMAGE_BUCKET;
  const op = encodeGcsPath(objectPath);
  if (!b || !op) throw new Error("missing_bucket_or_objectPath");

  const url = `https://storage.googleapis.com/${encodeURIComponent(b)}/${op}`;
  await putFileToUrl(url, file);
}

function buildDefaultListImageObjectPath(args: { listId: string; fileName: string }): string {
  const listId = s(args.listId);
  const fn = s(args.fileName) || "image";
  const safeName = fn.replace(/[^\w.\-]+/g, "_");
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return `lists/${listId}/${stamp}_${randomHex(6)}_${safeName}`;
}

export async function postListImageAfterCreate(args: {
  listId: string;
  image: PostListImageArgs;
}): Promise<{ image: ListImageDTO; list?: ListDTO; imageUrls?: string[] }> {
  const listId = s(args.listId);
  if (!listId) throw new Error("invalid_list_id");

  const imgArgs = args.image || {};
  const bucket = s(imgArgs.bucket) || LIST_IMAGE_BUCKET;

  // ---- file (optional) ----
  const file = takeFirstFile(imgArgs.fileInputRef);

  // objectPath
  let objectPath = s(imgArgs.objectPath);
  if (!objectPath && file) {
    objectPath = buildDefaultListImageObjectPath({ listId, fileName: file.name });
  }
  if (!objectPath) {
    throw new Error("missing_objectPath_for_listImage");
  }

  // ---- upload (best effort; if uploadUrl is provided, use it) ----
  if (file) {
    if (s(imgArgs.uploadUrl)) {
      await putFileToUrl(s(imgArgs.uploadUrl), file);
    } else {
      // public GCS へ PUT を試す（403なら bucket 側の権限設定が必要）
      await putFileToPublicGcs(bucket, objectPath, file);
    }
  }

  // ---- register image metadata to backend ----
  const imageId = randomHex(12);
  const createdAt = s(imgArgs.createdAt) || undefined;

  const createdBy = s(imgArgs.createdBy) || s(auth.currentUser?.uid) || "system";

  const image = await saveListImageFromGCSHTTP({
    listId,
    id: imageId,
    fileName: s(imgArgs.fileName) || (file ? s(file.name) : ""),
    bucket,
    objectPath,
    size: file ? file.size : 0,
    displayOrder: Number.isFinite(Number(imgArgs.displayOrder))
      ? Number(imgArgs.displayOrder)
      : 0,
    createdBy,
    createdAt,
  });

  // ---- set primary (optional) ----
  let list: ListDTO | undefined = undefined;
  if (imgArgs.setPrimary !== false) {
    try {
      list = await setListPrimaryImageHTTP({
        listId,
        imageId,
        updatedBy: s(auth.currentUser?.uid) || undefined,
      });
    } catch {
      // primary 設定に失敗しても image 登録は成功しているので、ここは握る
    }
  }

  return { image, list };
}

/**
 * ✅ list 作成（POST /lists）
 * ✅ その後、必要なら listImage を POST (/lists/{id}/images) して primary も設定する
 *
 * - 既存呼び出し互換のため、第2引数は optional
 */
export async function postCreateList(
  input: CreateListInput,
  opts?: { image?: PostListImageArgs },
): Promise<ListDTO> {
  // eslint-disable-next-line no-console
  console.log("[inventory/listCreateService] postCreateList (before validate)", {
    inventoryId: (input as any).inventoryId,
    title: (input as any).title,
    descriptionLen: String((input as any).description ?? "").length,
    decision: (input as any).decision,
    priceRowsCount: Array.isArray((input as any).priceRows) ? (input as any).priceRows.length : 0,
    priceRowsSample: Array.isArray((input as any).priceRows)
      ? (input as any).priceRows.slice(0, 5)
      : [],
    wantsImage: Boolean(opts?.image),
  });

  validateCreateListInput(input);

  const created = await createListHTTP(input);

  if (opts?.image) {
    const listId = getListIdFromListDTO(
      created,
      s((input as any)?.id) || s((input as any)?.inventoryId),
    );
    if (!listId) {
      throw new Error("created_list_missing_id");
    }

    try {
      const r = await postListImageAfterCreate({
        listId,
        image: { ...opts.image, setPrimary: opts.image.setPrimary !== false },
      });

      if (r.list) return r.list;
    } catch (e) {
      const msg = String(e instanceof Error ? e.message : e);
      // eslint-disable-next-line no-console
      console.log("[inventory/listCreateService] postCreateList: listImage post failed", {
        err: msg,
      });
      throw new Error(`list_image_post_failed: ${msg}`);
    }
  }

  return created;
}
