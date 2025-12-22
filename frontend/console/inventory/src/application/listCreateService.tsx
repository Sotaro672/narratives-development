// frontend/console/inventory/src/application/listCreateService.tsx
import type { RefObject } from "react";

import type { PriceRow } from "../../../list/src/presentation/hook/usePriceCard";

import {
  fetchListCreateDTO,
  type ListCreateDTO,
} from "../infrastructure/http/inventoryRepositoryHTTP";

// ✅ Firebase Auth（IDトークン / uid 取得）
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";

// ✅ list create (POST /lists) + listImage APIs（※ signed-url 発行も listRepositoryHTTP に寄せる）
import {
  createListHTTP,
  saveListImageFromGCSHTTP,
  setListPrimaryImageHTTP,
  issueListImageSignedUrlHTTP,
  type CreateListInput,
  type ListDTO,
  type SignedListImageUploadDTO,
} from "../../../list/src/infrastructure/http/listRepositoryHTTP";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

/**
 * ✅ 方針A: inventoryId は "pb__tb" をそのまま通す（splitしない）
 */
function normalizeInventoryId(v: unknown): string {
  return s(v);
}

/**
 * ✅ listId は backend が採番する想定。
 * "__" を含んでいても正当なIDになり得るので split しない。
 */
function normalizeListId(v: unknown): string {
  return s(v);
}

// ✅ Hook 側で使う Ref 型（useRef<HTMLInputElement | null>(null) を許容）
export type ImageInputRef = RefObject<HTMLInputElement | null>;

export type ListCreateRouteParams = {
  inventoryId?: string;          // 期待値: "pb__tb"
  productBlueprintId?: string;   // optional
  tokenBlueprintId?: string;     // optional
};

export type ResolvedListCreateParams = {
  inventoryId: string;          // ✅ 常に "pb__tb" を保持
  productBlueprintId: string;
  tokenBlueprintId: string;
  raw: ListCreateRouteParams;
};

/**
 * ✅ param 解決
 * - inventoryId が来ていればそれを優先（pb__tbを維持）
 * - inventoryId が無ければ pbId + tbId から pb__tb を合成
 * - inventoryId しか無い場合は pb/tb を補完（抽出のために split はするが inventoryId は変えない）
 */
export function resolveListCreateParams(
  raw: ListCreateRouteParams,
): ResolvedListCreateParams {
  const inv = normalizeInventoryId(raw?.inventoryId);
  const pbRaw = s(raw?.productBlueprintId);
  const tbRaw = s(raw?.tokenBlueprintId);

  // inventoryId が無いなら pb/tb から合成
  const inventoryId = inv || (pbRaw && tbRaw ? `${pbRaw}__${tbRaw}` : "");

  // pb/tb が無いなら inventoryId から補完（※抽出のための split）
  let productBlueprintId = pbRaw;
  let tokenBlueprintId = tbRaw;
  if ((!productBlueprintId || !tokenBlueprintId) && inventoryId.includes("__")) {
    const parts = inventoryId.split("__");
    const pb = s(parts[0]);
    const tb = s(parts[1]);
    if (!productBlueprintId) productBlueprintId = pb;
    if (!tokenBlueprintId) tokenBlueprintId = tb;
  }

  return {
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,
    raw,
  };
}

export function computeListCreateTitle(inventoryId: string): string {
  return inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";
}

export function canFetchListCreate(p: ResolvedListCreateParams): boolean {
  // ✅ 方針A: inventoryId（pb__tb）があれば取得できる
  return Boolean(p.inventoryId);
}

export function buildListCreateFetchInput(p: ResolvedListCreateParams): {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
} {
  // ✅ 方針A: backend は inventoryId（pb__tb）を期待
  return {
    inventoryId: p.inventoryId || undefined,
    productBlueprintId: undefined,
    tokenBlueprintId: undefined,
  };
}

export function getInventoryIdFromDTO(
  dto: ListCreateDTO | null | undefined,
): string {
  return normalizeInventoryId((dto as any)?.inventoryId ?? (dto as any)?.InventoryID);
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
  const id = normalizeInventoryId(inventoryId);
  if (!id) return "/inventory/list/create";
  // ✅ pb__tb をそのまま URL に入れる
  return `/inventory/list/create/${encodeURIComponent(id)}`;
}

export function buildBackPath(p: ResolvedListCreateParams): string {
  if (p.productBlueprintId && p.tokenBlueprintId) {
    return buildInventoryDetailPath(p.productBlueprintId, p.tokenBlueprintId);
  }
  // pb/tb が補完できない場合は一覧へ
  return "/inventory";
}

export function buildAfterCreatePath(p: ResolvedListCreateParams): string {
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
    const rgb = r?.rgb ?? r?.RGB;
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

export type PriceRowEx = PriceRow & {
  modelId: string; // ✅ 必須
};

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

export function normalizeCreateListPriceRows(rows: any[]): CreateListPriceRow[] {
  const arr = Array.isArray(rows) ? rows : [];
  return arr.map((r) => {
    const modelId = s((r as any)?.modelId ?? (r as any)?.ModelID);
    const price = toNumberOrNull((r as any)?.price);
    return { modelId, price };
  });
}

export function buildCreateListInput(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;
}): CreateListInput {
  const title = s(args.listingTitle);
  const desc = s(args.description);

  const priceRows = normalizeCreateListPriceRows(args.priceRows);

  return {
    // ✅ 最重要: inventoryId(pb__tb) をそのまま送る
    inventoryId: normalizeInventoryId(args.params.inventoryId) || undefined,
    title,
    description: desc,
    decision: args.decision,
    assigneeId: s(args.assigneeId) || undefined,
    priceRows: priceRows as any,
  } as CreateListInput;
}

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

  const hasPositivePrice = rows.some((r: any) => {
    const n = toNumberOrNull(r?.price);
    return n !== null && n > 0;
  });
  if (!hasPositivePrice) {
    throw new Error("価格を入力してください。（0 円は指定できません）");
  }

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

function getListIdFromListDTO(dto: ListDTO, fallback = ""): string {
  const raw =
    s((dto as any)?.id) ||
    s((dto as any)?.ID) ||
    s((dto as any)?.listId) ||
    s((dto as any)?.ListID) ||
    s(fallback);

  return normalizeListId(raw);
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

/**
 * ✅ 複数画像を Policy A（signed-url）でアップロード→メタ登録→primary 設定
 */
export async function uploadListImagesPolicyA(args: {
  listId: string;
  files: File[];
  mainImageIndex: number;
  createdBy?: string;
}): Promise<{ registered: Array<{ imageId: string; displayOrder: number }>; primaryImageId?: string }> {
  const listId = normalizeListId(args.listId);
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

    const signed: SignedListImageUploadDTO = await issueListImageSignedUrlHTTP({
      listId,
      fileName: file.name,
      contentType: file.type || "application/octet-stream",
      size: file.size || 0,
      displayOrder: i,
    });

    const objectPath = s(signed.objectPath);
    const signedUrl = s(signed.signedUrl);
    const bucket = s(signed.bucket);

    if (!objectPath || !signedUrl) {
      throw new Error("signed_url_response_invalid");
    }

    await putFileToSignedUrl({ signedUrl, file });

    await saveListImageFromGCSHTTP({
      listId,
      id: objectPath,
      fileName: s(file.name),
      bucket,
      objectPath,
      size: Number(file.size || 0),
      displayOrder: i,
      createdBy: uid,
      createdAt: now,
    });

    registered.push({ imageId: objectPath, displayOrder: i });
  }

  const primary =
    registered.find((x) => x.displayOrder === mainImageIndex) || registered[0];

  if (primary?.imageId) {
    await setListPrimaryImageHTTP({
      listId,
      imageId: primary.imageId,
      updatedBy: uid,
      now,
    } as any);
  }

  return { registered, primaryImageId: primary?.imageId };
}

// ============================================================
// ✅ list 作成（POST /lists） + 画像（Policy A）
// ============================================================

export async function createListWithImages(args: {
  params: ResolvedListCreateParams;
  listingTitle: string;
  description: string;
  priceRows: any[];
  decision: "list" | "hold";
  assigneeId?: string;

  images?: File[];
  mainImageIndex?: number;
}): Promise<ListDTO> {
  const images = Array.isArray(args.images) ? args.images : [];
  const mainImageIndex = Number.isFinite(Number(args.mainImageIndex))
    ? Number(args.mainImageIndex)
    : 0;

  // 1) build + validate
  const input = buildCreateListInput({
    params: args.params, // ✅ inventoryId(pb__tb) を保持
    listingTitle: args.listingTitle,
    description: args.description,
    priceRows: args.priceRows,
    decision: args.decision,
    assigneeId: args.assigneeId,
  });

  validateCreateListInput(input);

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
    await uploadListImagesPolicyA({
      listId,
      files: images,
      mainImageIndex,
      createdBy: s(auth.currentUser?.uid) || undefined,
    });
  }

  return created;
}
