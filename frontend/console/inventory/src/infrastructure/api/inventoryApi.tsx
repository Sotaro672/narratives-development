// frontend/console/inventory/src/infrastructure/api/inventoryApi.tsx

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

/**
 * Backend base URL
 * - .env の VITE_BACKEND_BASE_URL を優先
 * - 未設定時は Cloud Run の固定 URL を利用
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) throw new Error("Not authenticated");
  const token = await user.getIdToken();
  if (!token) throw new Error("Failed to acquire ID token");
  return token;
}

async function requestJsonOrThrow(path: string): Promise<any> {
  const token = await getIdTokenOrThrow();

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`request failed: ${res.status} ${res.statusText} ${text}`);
  }

  return await res.json();
}

function s(v: unknown): string {
  return String(v ?? "").trim();
}

// ---------------------------------------------------------
// Inventory APIs (raw JSON)
// ---------------------------------------------------------

/** GET /inventory */
export async function getInventoryListRaw(): Promise<any> {
  return await requestJsonOrThrow(`/inventory`);
}

/** GET /product-blueprints/{id} */
export async function getProductBlueprintRaw(productBlueprintId: string): Promise<any> {
  const pbId = s(productBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");
  return await requestJsonOrThrow(`/product-blueprints/${encodeURIComponent(pbId)}`);
}

/** GET /product-blueprints/printed */
export async function getPrintedProductBlueprintsRaw(): Promise<any> {
  return await requestJsonOrThrow(`/product-blueprints/printed`);
}

/** GET /inventory/ids?productBlueprintId=...&tokenBlueprintId=... */
export async function getInventoryIDsByProductAndTokenRaw(input: {
  productBlueprintId: string;
  tokenBlueprintId: string;
}): Promise<any> {
  const pbId = s(input.productBlueprintId);
  const tbId = s(input.tokenBlueprintId);
  if (!pbId) throw new Error("productBlueprintId is empty");
  if (!tbId) throw new Error("tokenBlueprintId is empty");

  const path = `/inventory/ids?productBlueprintId=${encodeURIComponent(
    pbId,
  )}&tokenBlueprintId=${encodeURIComponent(tbId)}`;

  return await requestJsonOrThrow(path);
}

/** GET /token-blueprints/{tokenBlueprintId}/patch */
export async function getTokenBlueprintPatchRaw(tokenBlueprintId: string): Promise<any> {
  const tbId = s(tokenBlueprintId);
  if (!tbId) throw new Error("tokenBlueprintId is empty");
  return await requestJsonOrThrow(
    `/token-blueprints/${encodeURIComponent(tbId)}/patch`,
  );
}

/**
 * GET
 * - /inventory/list-create/:inventoryId
 * - /inventory/list-create/:productBlueprintId/:tokenBlueprintId
 */
export async function getListCreateRaw(input: {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
}): Promise<any> {
  const inventoryId = s(input.inventoryId);
  const productBlueprintId = s(input.productBlueprintId);
  const tokenBlueprintId = s(input.tokenBlueprintId);

  let path = "";
  if (inventoryId) {
    path = `/inventory/list-create/${encodeURIComponent(inventoryId)}`;
  } else if (productBlueprintId && tokenBlueprintId) {
    path =
      `/inventory/list-create/${encodeURIComponent(
        productBlueprintId,
      )}/${encodeURIComponent(tokenBlueprintId)}`;
  } else {
    throw new Error("missing params");
  }

  return await requestJsonOrThrow(path);
}

/** GET /inventory/{inventoryId} */
export async function getInventoryDetailRaw(inventoryId: string): Promise<any> {
  const id = s(inventoryId);
  if (!id) throw new Error("inventoryId is empty");
  return await requestJsonOrThrow(`/inventory/${encodeURIComponent(id)}`);
}
