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

/**
 * /orders/{id} の items 1件
 * - backend が inventoryId -> pb/tb だけでなく name も返すようになった前提
 * - backend が modelId(=variationID) から size/color/rgb/modelNumber も返すようになった前提
 */
export type OrderItemDTO = {
  modelId?: string;

  // ✅ NEW: modelId(variationID) -> variation fields (UI表示用)
  size?: string;
  color?: string;
  rgb?: string;
  modelNumber?: string;

  // backward-compat (UI が使わなくても返ってくる可能性がある)
  inventoryId?: string;

  // resolved from inventoryId
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  // ✅ resolved names
  productName?: string;
  tokenName?: string;

  listId?: string;

  // ✅ listId -> readableId（UIで表示用）
  // backendが listReadableId もしくは互換キーで返す場合に拾えるようにする
  listReadableId?: string;

  qty?: number;
  price?: number;

  transferred: boolean;
  transferredAt?: string; // RFC3339(UTC)

  [k: string]: any;
};

export type Order = {
  id: string;

  // ✅ userId ではなく userName（lastName→firstName）
  userName?: string;

  avatarId: string;

  // ✅ avatarId -> avatarName（UI表示用）
  avatarName?: string;

  cartId: string;
  paid: boolean;
  createdAt: string; // RFC3339
  shippingSnapshot?: any;
  billingSnapshot?: any;

  // ✅ any[] ではなく DTO を持つ（画面で productName/tokenName/size/color/rgb/modelNumber を拾える）
  items?: OrderItemDTO[];
};

/**
 * /orders/items の 1行DTO（フラット）
 * - backend OrderManagementQuery が company boundary を通した items のみ返す想定
 * - backend が modelId(=variationID) から size/color/rgb/modelNumber も返すようになった前提
 */
export type OrderItemInventoryRowDTO = {
  orderId: string;

  // ✅ userId ではなく userName（lastName→firstName）
  userName?: string;

  avatarId?: string;

  // ✅ avatarId -> avatarName（UI表示用）
  avatarName?: string;

  cartId?: string;

  paid: boolean;
  createdAt?: string; // RFC3339(UTC)

  inventoryId: string;

  // resolved from inventoryId
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  // ✅ resolved names
  productName?: string;
  tokenName?: string;

  listId?: string;

  // ✅ listId -> readableId（UIで表示用）
  listReadableId?: string;

  modelId?: string;

  // ✅ NEW: model fields (variation)
  size?: string;
  color?: string;
  rgb?: string;
  modelNumber?: string;

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

  // ✅ フィルタは引き続き userId を使う（APIの検索条件が userId 前提のため）
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
    t.startsWith("<!DOCTYPE html") ||
    t.startsWith("<html") ||
    t.startsWith("<!--")
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

// ✅ best-effort: backend のキー揺れを吸収して listReadableId / avatarName / userName / model fields に寄せる
function normalizeInPlace(obj: any): void {
  if (!obj || typeof obj !== "object") return;

  // PageResult<T> / Order
  if (Array.isArray(obj.items)) {
    for (const it of obj.items) normalizeInPlace(it);
  }

  // Order.items
  if (Array.isArray(obj.items)) {
    for (const it of obj.items) normalizeInPlace(it);
  }

  // ----------------------------
  // listReadableId
  // ----------------------------
  if (obj.listReadableId == null || String(obj.listReadableId).trim() === "") {
    const candidates = [
      obj.listReadableID, // camel with ID
      obj.listReadableId, // itself
      obj.readableId, // generic readableId
      obj.readableID, // generic readableID
      obj.list_readable_id, // snake
    ];
    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.listReadableId = v.trim();
        break;
      }
    }
  }

  // ----------------------------
  // avatarName
  // ----------------------------
  if (obj.avatarName == null || String(obj.avatarName).trim() === "") {
    const candidates = [
      obj.avatar_name, // snake
      obj.avatarName, // itself
      obj.avatar_name_jp, // if ever
    ];
    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.avatarName = v.trim();
        break;
      }
    }
  }

  // ----------------------------
  // userName（lastName→firstName）
  // ----------------------------
  if (obj.userName == null || String(obj.userName).trim() === "") {
    const candidates = [
      obj.user_name, // snake
      obj.userName, // itself
      obj.user_name_jp, // if ever
    ];
    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.userName = v.trim();
        break;
      }
    }
  }

  // ----------------------------
  // model fields: size / color / rgb / modelNumber
  // ----------------------------
  if (obj.size == null || String(obj.size).trim() === "") {
    const candidates = [obj.Size, obj.size, obj.model_size, obj.modelSize];
    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.size = v.trim();
        break;
      }
    }
  }

  if (obj.modelNumber == null || String(obj.modelNumber).trim() === "") {
    const candidates = [
      obj.modelNumber,
      obj.model_number,
      obj.modelNo,
      obj.model_no,
      obj.ModelNumber,
    ];
    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.modelNumber = v.trim();
        break;
      }
    }
  }

  // color は "string(色名)" として受け取りたい
  // backend が {color:{name:"..",rgb:".."}} を返す可能性もあるので吸収
  if (obj.color == null || String(obj.color).trim() === "") {
    const candidates: any[] = [
      obj.color,
      obj.color_name,
      obj.colorName,
      obj.Color, // sometimes
      obj.ColorName,
    ];

    // object pattern: color: { name, rgb }
    if (obj.color && typeof obj.color === "object") {
      candidates.unshift(obj.color.name, obj.color.Name);
      // rgb も同時に拾える
      if (obj.rgb == null || String(obj.rgb).trim() === "") {
        const rv = obj.color.rgb ?? obj.color.RGB;
        if (typeof rv === "string" && rv.trim() !== "") obj.rgb = rv.trim();
      }
    }

    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.color = v.trim();
        break;
      }
    }
  }

  if (obj.rgb == null || String(obj.rgb).trim() === "") {
    const candidates: any[] = [
      obj.rgb,
      obj.RGB,
      obj.colorRgb,
      obj.color_rgb,
    ];

    // object pattern: color: { rgb }
    if (obj.color && typeof obj.color === "object") {
      candidates.unshift(obj.color.rgb, obj.color.RGB);
    }

    for (const v of candidates) {
      if (typeof v === "string" && v.trim() !== "") {
        obj.rgb = v.trim();
        break;
      }
    }
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

  const data = (await res.json()) as any;

  // ✅ listReadableId / avatarName / userName / model fields を拾えるように正規化（best-effort）
  normalizeInPlace(data);

  return data as T;
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
        userId: params.userId, // ✅ フィルタ用
        avatarId: params.avatarId,
        cartId: params.cartId,

        createdFrom: params.createdFrom,
        createdTo: params.createdTo,
      });

      const url = buildUrl(`/orders/items${qs}`);
      // ✅ ここで受け取る DTO に
      // productBlueprintId/tokenBlueprintId/productName/tokenName/listReadableId/avatarName/userName
      // + size/color/rgb/modelNumber が含まれる
      return requestJSON<PageResult<OrderItemInventoryRowDTO>>(fetcher, url, {
        method: "GET",
      });
    },

    async listDistinctInventoryIds(params: OrderListParams = {}) {
      const qs = buildQuery({
        page: params.page ?? 1,
        perPage: params.perPage ?? 200,

        id: params.id,
        userId: params.userId, // ✅ フィルタ用
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
