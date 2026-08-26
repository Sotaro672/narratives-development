// frontend/console/shell/src/features/order/infrastructure/repository.ts

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import { API_BASE } from "../../../shared/http/apiBase";
import type { PageParams, PageResult } from "../../../shared/types/common/common";

export type ShippingSnapshot = {
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
  country: string;
};

export type PaymentMethodSnapshot = {
  customerId: string;
  brand: string;
  last4: string;
  expMonth: number;
  expYear: number;
  cardholderName: string;
  isDefault: boolean;
};

export type OrderItemType = "list" | "resale";

export type OrderDetailItemDTO = {
  type: OrderItemType;
  modelId?: string;
  inventoryId?: string;
  listId?: string;
  resaleId?: string;
  productId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  brandId?: string;
  productName: string;
  tokenName: string;
  listReadableId?: string;
  categoryId: string;
  categoryCode: string;
  categoryNameJa: string;
  categoryNameEn: string;
  categoryKind: string;
  categoryPath: string[];
  categoryFields: Record<string, unknown>;
  kind: string;
  modelNumber: string;
  size: string;
  color: string;
  rgb?: number;
  volumeValue?: number;
  volumeUnit: string;
  qty: number;
  price: number;
  isCancelled: boolean;
  isDispatched: boolean;
  transferred: boolean;
  transferredAt?: string;
};

export type OrderDetailDTO = {
  id: string;
  userId: string;
  avatarId: string;
  cartId: string;
  userName: string;
  email: string;
  paid: boolean;
  createdAt: string;
  shippingAmount: number;
  consumptionTax: number;
  shippingSnapshot: ShippingSnapshot;
  paymentMethodSnapshot: PaymentMethodSnapshot;
  items: OrderDetailItemDTO[];
};

/**
 * GET /orders/items の1行DTO。
 * backend OrderManagementQuery の response をそのまま受け取る。
 */
export type OrderItemInventoryRowDTO = {
  orderId: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  userName?: string;
  paid: boolean;
  createdAt?: string;
  inventoryId: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
  productName?: string;
  tokenName?: string;
  listReadableId?: string;
  categoryId?: string;
  categoryCode?: string;
  categoryNameJa?: string;
  categoryNameEn?: string;
  categoryKind?: string;
  categoryPath?: string[];
  categoryFields?: Record<string, unknown>;
  modelId?: string;
  kind?: string;
  modelNumber?: string;
  size?: string;
  color?: string;
  rgb?: string;
  volumeValue?: number;
  volumeUnit?: string;
  qty?: number;
  price?: number;
  isCancelled: boolean;
  isDispatched: boolean;
  isReturnRequested: boolean;
  transferred: boolean;
  transferredAt?: string;
};

export type OrderActionRequiredCountResult = {
  count: number;
};

export type OrderListParams = PageParams & {
  id?: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  createdFrom?: string;
  createdTo?: string;
};

function buildQuery(
  params: Record<string, string | number | boolean | undefined>,
): string {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined) continue;

    const stringValue = String(value);
    if (!stringValue) continue;

    searchParams.set(key, stringValue);
  }

  const query = searchParams.toString();
  return query ? `?${query}` : "";
}

async function readErrorMessage(response: Response): Promise<string> {
  const contentType = response.headers.get("content-type") ?? "";

  try {
    if (contentType.includes("application/json")) {
      const body: unknown = await response.json();

      if (
        body &&
        typeof body === "object" &&
        "error" in body &&
        typeof body.error === "string"
      ) {
        return body.error;
      }

      return `${response.status} ${response.statusText}`;
    }

    const text = await response.text();
    return text
      ? text.slice(0, 200)
      : `${response.status} ${response.statusText}`;
  } catch {
    return `${response.status} ${response.statusText}`;
  }
}

async function requestJSON<T>(
  url: string,
  init?: RequestInit,
): Promise<T> {
  const authHeaders = await getAuthHeaders();
  const headers = new Headers(init?.headers);

  headers.set("Accept", "application/json");

  if (!headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  for (const [key, value] of Object.entries(authHeaders)) {
    if (!headers.has(key)) {
      headers.set(key, value);
    }
  }

  const response = await fetch(url, {
    ...init,
    headers,
  });

  if (!response.ok) {
    throw new Error(await readErrorMessage(response));
  }

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    const text = await response.text().catch(() => "");

    throw new Error(
      `API returned non-JSON response. url=${url} content-type=${contentType}${
        text ? ` body=${text.slice(0, 200)}` : ""
      }`,
    );
  }

  return response.json() as Promise<T>;
}

export interface OrderRepository {
  getById(id: string): Promise<OrderDetailDTO>;
  dispatch(id: string): Promise<OrderDetailDTO>;
  listItemInventoryRows(
    params?: OrderListParams,
  ): Promise<PageResult<OrderItemInventoryRowDTO>>;
  countActionRequired(): Promise<OrderActionRequiredCountResult>;
}

export function createOrderRepository(): OrderRepository {
  const resolvedBaseUrl = API_BASE.replace(/\/+$/g, "");

  const buildUrl = (path: string): string => {
    const normalizedPath = path.replace(/^\/+/g, "");
    return `${resolvedBaseUrl}/${normalizedPath}`;
  };

  return {
    async getById(id: string): Promise<OrderDetailDTO> {
      if (!id) {
        throw new Error("id is required");
      }

      return requestJSON<OrderDetailDTO>(
        buildUrl(`/orders/${encodeURIComponent(id)}`),
        { method: "GET" },
      );
    },

    async dispatch(id: string): Promise<OrderDetailDTO> {
      if (!id) {
        throw new Error("id is required");
      }

      return requestJSON<OrderDetailDTO>(
        buildUrl(`/orders/${encodeURIComponent(id)}/dispatch`),
        { method: "PATCH" },
      );
    },

    async listItemInventoryRows(
      params: OrderListParams = {},
    ): Promise<PageResult<OrderItemInventoryRowDTO>> {
      const query = buildQuery({
        page: params.page ?? 1,
        perPage: params.perPage ?? 20,
        id: params.id,
        userId: params.userId,
        avatarId: params.avatarId,
        cartId: params.cartId,
        createdFrom: params.createdFrom,
        createdTo: params.createdTo,
      });

      return requestJSON<PageResult<OrderItemInventoryRowDTO>>(
        buildUrl(`/orders/items${query}`),
        { method: "GET" },
      );
    },

    async countActionRequired(): Promise<OrderActionRequiredCountResult> {
      return requestJSON<OrderActionRequiredCountResult>(
        buildUrl("/orders/action-required-count"),
        { method: "GET" },
      );
    },
  };
}