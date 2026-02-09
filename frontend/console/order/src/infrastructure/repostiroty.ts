// frontend/console/order/src/infrastructure/repostiroty.ts
// NOTE: backend console router.go に合わせて /orders を叩くリポジトリ
// - GET /orders/{id}
// - GET /orders/items
// - GET /orders/inventory-ids

import { getAuthHeaders } from "../../../shell/src/shared/http/authHeaders";
import { API_BASE, buildConsoleUrl } from "../../../shell/src/shared/http/apiBase";

export type SortOrder = "asc" | "desc";

export type PageResult<T> = {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

export type Order = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;
  paid: boolean;
  createdAt: string; // RFC3339
  shippingSnapshot?: any;
  billingSnapshot?: any;
  items?: any[];
};

export type OrderItemInventoryRowDTO = {
  orderId: string;

  userId?: string;
  avatarId?: string;
  cartId?: string;

  paid: boolean;
  createdAt?: string; // RFC3339(UTC)

  inventoryId: string;

  // ✅ NEW: backend OrderManagementQuery が inventoryId から解決して返す
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  listId?: string;
  modelId?: string;
  qty?: number;
  price?: number;

  transferred: boolean;
  transferredAt?: string; // RFC3339(UTC)
};

export type InventoryIDDTO = {
  inventoryId: string;
};

export type OrderListParams = {
  page?: number;
  perPage?: number;

  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;

  createdFrom?: string; // RFC3339
  createdTo?: string; // RFC3339
};

export type RepositoryConfig = {
  /**
   * テスト用に fetch を差し替える
   */
  fetcher?: typeof fetch;

  /**
   * 例外的に API base を差し替えたい場合のみ使用
   * （通常は shell/shared/http/apiBase.ts の API_BASE を使う）
   */
  baseUrl?: string;
};

function buildQuery(
  params: Record<string, string | number | boolean | undefined>,
): string {
  const sp = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined) continue;
    const s = String(v).trim();
    if (s === "") continue;
    sp.set(k, s);
  }
  const qs = sp.toString();
  return qs ? `?${qs}` : "";
}

function isLikelyHtml(text: string): boolean {
  const t = (text ?? "").trimStart();
  return (
    t.startsWith("<!DOCTYPE html") || t.startsWith("<html") || t.startsWith("<!--")
  );
}

async function readErrorMessage(res: Response): Promise<string> {
  const ct = res.headers.get("content-type") ?? "";
  try {
    if (ct.includes("application/json")) {
      const j: any = await res.json();
      if (j?.error) return String(j.error);
      return `${res.status} ${res.statusText}`;
    }
    const t = await res.text();
    if (isLikelyHtml(t)) {
      return `API returned HTML (not JSON). Check API base/rewrite/auth. status=${res.status}`;
    }
    return t ? t.slice(0, 200) : `${res.status} ${res.statusText}`;
  } catch {
    return `${res.status} ${res.statusText}`;
  }
}

async function requestJSON<T>(
  fetcher: typeof fetch,
  url: string,
  init?: RequestInit,
): Promise<T> {
  // ✅ shell共通の認証ヘッダを導入（ここに “infrastructure” を集約）
  const auth = await getAuthHeaders();

  // headers merge（caller headers があれば尊重しつつ auth を足す）
  const headers = new Headers(init?.headers ?? {});
  headers.set("Accept", "application/json");
  if (!headers.has("Content-Type")) headers.set("Content-Type", "application/json");

  // auth headers（Authorization 等）を付与
  for (const [k, v] of Object.entries(auth)) {
    if (!headers.has(k)) headers.set(k, v);
  }

  const res = await fetcher(url, { ...init, headers });

  if (!res.ok) {
    const msg = await readErrorMessage(res);
    throw new Error(msg);
  }

  // ✅ 200でもHTMLが返る事故を検出してわかりやすく落とす
  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    const t = await res.text();
    if (isLikelyHtml(t)) {
      throw new Error(
        `API returned HTML (not JSON). Most likely wrong base URL or hosting rewrite. url=${url}`,
      );
    }
    throw new Error(`API returned non-JSON response. url=${url} content-type=${ct}`);
  }

  return (await res.json()) as T;
}

export interface OrderRepository {
  getById(id: string): Promise<Order>;

  listItemInventoryRows(
    params?: OrderListParams,
  ): Promise<PageResult<OrderItemInventoryRowDTO>>;

  listDistinctInventoryIds(
    params?: OrderListParams,
  ): Promise<PageResult<InventoryIDDTO>>;
}

/**
 * createOrderRepository
 * - baseUrl 未指定なら shell の API_BASE を使用（= 絶対URL）
 * - 認証ヘッダ付与は requestJSON 内で getAuthHeaders() に集約
 */
export function createOrderRepository(cfg: RepositoryConfig = {}): OrderRepository {
  const fetcher = cfg.fetcher ?? fetch;
  const baseUrl = (cfg.baseUrl ?? API_BASE).replace(/\/+$/g, "");

  const buildUrl = (path: string): string => {
    // buildConsoleUrl は env 正規化済み origin を使う。
    // cfg.baseUrl が指定された場合はそれを優先。
    if (cfg.baseUrl) {
      const p = String(path ?? "").replace(/^\/+/g, "");
      return `${baseUrl}/${p}`;
    }
    return buildConsoleUrl(path);
  };

  return {
    async getById(id: string) {
      const orderId = String(id ?? "").trim();
      if (!orderId) throw new Error("id is required");

      const url = buildUrl(`/orders/${encodeURIComponent(orderId)}`);
      return requestJSON<Order>(fetcher, url, { method: "GET" });
    },

    async listItemInventoryRows(params: OrderListParams = {}) {
      const qs = buildQuery({
        page: params.page ?? 1,
        perPage: params.perPage ?? 20,

        id: params.id,
        userId: params.userId,
        avatarId: params.avatarId,
        cartId: params.cartId,

        createdFrom: params.createdFrom,
        createdTo: params.createdTo,
      });

      const url = buildUrl(`/orders/items${qs}`);
      // ✅ ここで受け取る DTO に productBlueprintId/tokenBlueprintId が含まれる
      return requestJSON<PageResult<OrderItemInventoryRowDTO>>(fetcher, url, {
        method: "GET",
      });
    },

    async listDistinctInventoryIds(params: OrderListParams = {}) {
      const qs = buildQuery({
        page: params.page ?? 1,
        perPage: params.perPage ?? 200,

        id: params.id,
        userId: params.userId,
        avatarId: params.avatarId,
        cartId: params.cartId,

        createdFrom: params.createdFrom,
        createdTo: params.createdTo,
      });

      const url = buildUrl(`/orders/inventory-ids${qs}`);
      return requestJSON<PageResult<InventoryIDDTO>>(fetcher, url, {
        method: "GET",
      });
    },
  };
}
