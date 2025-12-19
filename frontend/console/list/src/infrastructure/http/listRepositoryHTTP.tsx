// frontend/console/list/src/infrastructure/http/listRepositoryHTTP.tsx

// Firebase Auth から ID トークンを取得
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

// eslint-disable-next-line no-console
console.log("[list/listRepositoryHTTP] API_BASE resolved =", API_BASE, {
  ENV_BASE,
  usingFallback: !ENV_BASE,
});

/**
 * ===========
 * Types
 * ===========
 * ※ backend の List エンティティに完全一致しなくてもOK（必要な分だけ）
 */

export type CreateListInput = {
  // ルート（inventory/list/create から作成する想定）
  inventoryId?: string;

  // UI 入力
  title: string;
  description: string;

  // PriceCard の rows（必要に応じて拡張）
  // 例: [{ size:"S", color:"Red", stock:10, price:1000, rgb:123 }]
  priceRows?: Array<{
    size: string;
    color: string;
    stock: number;
    price: number | null;
    rgb?: number | null;
  }>;

  // 画面の「出品｜保留」
  decision?: "list" | "hold";

  // 担当者など（必要に応じて）
  assigneeId?: string;

  // 作成者など（バックエンドで auth から取るなら省略可）
  createdBy?: string;
};

export type ListDTO = Record<string, any>;
export type ListAggregateDTO = Record<string, any>;
export type ListImageDTO = Record<string, any>;

/**
 * ===========
 * Internal helpers
 * ===========
 */

async function getIdToken(): Promise<string> {
  const u = auth.currentUser;
  if (!u) {
    throw new Error("not_authenticated");
  }
  return await u.getIdToken();
}

async function requestJSON<T>(args: {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  body?: unknown;
}): Promise<T> {
  const token = await getIdToken();
  const url = `${API_BASE}${args.path.startsWith("/") ? "" : "/"}${args.path}`;

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] request", {
    method: args.method,
    url,
    body: args.body,
  });

  const res = await fetch(url, {
    method: args.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: args.body === undefined ? undefined : JSON.stringify(args.body),
  });

  const text = await res.text();
  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    json = { raw: text };
  }

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] response", {
    method: args.method,
    url,
    status: res.status,
    ok: res.ok,
    keys: json && typeof json === "object" ? Object.keys(json) : [],
    body: json,
  });

  if (!res.ok) {
    const msg =
      (json && typeof json === "object" && (json.error || json.message)) ||
      `http_error_${res.status}`;
    throw new Error(String(msg));
  }

  return json as T;
}

/**
 * ===========
 * API
 * ===========
 *
 * Backend:
 * - POST /lists            (create)
 * - GET  /lists/{id}       (detail)
 * - GET  /lists/{id}/aggregate
 * - GET  /lists/{id}/images
 * - POST /lists/{id}/images
 * - PUT|POST|PATCH /lists/{id}/primary-image
 */

/**
 * ✅ Create list
 * POST /lists
 */
export async function createListHTTP(input: CreateListInput): Promise<ListDTO> {
  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] createListHTTP payload", {
    input,
    titleLen: String(input?.title ?? "").length,
    descriptionLen: String(input?.description ?? "").length,
    priceRowsCount: Array.isArray(input?.priceRows) ? input.priceRows.length : 0,
  });

  return await requestJSON<ListDTO>({
    method: "POST",
    path: "/lists",
    body: input,
  });
}

/**
 * GET /lists/{id}
 */
export async function fetchListByIdHTTP(listId: string): Promise<ListDTO> {
  const id = String(listId ?? "").trim();
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListDTO>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}`,
  });
}

/**
 * GET /lists/{id}/aggregate
 */
export async function fetchListAggregateHTTP(listId: string): Promise<ListAggregateDTO> {
  const id = String(listId ?? "").trim();
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
  const id = String(listId ?? "").trim();
  if (!id) throw new Error("invalid_list_id");

  return await requestJSON<ListImageDTO[]>({
    method: "GET",
    path: `/lists/${encodeURIComponent(id)}/images`,
  });
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
  const listId = String(args.listId ?? "").trim();
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

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] saveListImageFromGCSHTTP payload", payload);

  return await requestJSON<ListImageDTO>({
    method: "POST",
    path: `/lists/${encodeURIComponent(listId)}/images`,
    body: payload,
  });
}

/**
 * PUT|POST|PATCH /lists/{id}/primary-image
 */
export async function setListPrimaryImageHTTP(args: {
  listId: string;
  imageId: string;
  updatedBy?: string;
  now?: string; // RFC3339 optional
}): Promise<ListDTO> {
  const listId = String(args.listId ?? "").trim();
  if (!listId) throw new Error("invalid_list_id");

  const payload = {
    imageId: String(args.imageId ?? "").trim(),
    updatedBy: args.updatedBy ? String(args.updatedBy).trim() : undefined,
    now: args.now ? String(args.now).trim() : undefined,
  };

  // eslint-disable-next-line no-console
  console.log("[list/listRepositoryHTTP] setListPrimaryImageHTTP payload", payload);

  return await requestJSON<ListDTO>({
    method: "PUT",
    path: `/lists/${encodeURIComponent(listId)}/primary-image`,
    body: payload,
  });
}
