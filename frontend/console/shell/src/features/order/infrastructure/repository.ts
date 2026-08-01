// frontend/console/shell/src/features/order/infrastructure/repository.ts

// NOTE: backend console router.go に合わせて /orders を叩くリポジトリ
// - GET /orders/{id}
// - GET /orders/items

import { getAuthHeaders } from "../../../shared/http/authHeaders";
import { API_BASE } from "../../../shared/http/apiBase";

import type {
  PageParams,
  PageResult,
} from "../../../shared/types/common/common";

export type ShippingSnapshot = {
  zipCode?: string;
  state?: string;
  city?: string;
  street?: string;
  street2?: string;
  country?: string;
};

export type Order = {
  id: string;
  userId?: string;
  avatarId?: string;
  cartId?: string;
  userName?: string;
  avatarName?: string;
  paid: boolean;
  createdAt?: string;
  shippingSnapshot?: ShippingSnapshot;
};

/**
 * /orders/items の1行DTO。
 *
 * 正とするレスポンス:
 * {
 *   orderId,
 *   userId,
 *   avatarId,
 *   cartId,
 *   avatarName,
 *   paid,
 *   createdAt,
 *   inventoryId,
 *   productBlueprintId,
 *   tokenBlueprintId,
 *   productName,
 *   tokenName,
 *   listReadableId,
 *   modelId,
 *   kind,
 *   modelNumber,
 *   size?,
 *   color?,
 *   rgb?,
 *   volumeValue?,
 *   volumeUnit?,
 *   qty,
 *   price,
 *   transferred,
 *   transferredAt?
 * }
 */
export type OrderItemInventoryRowDTO = {
  orderId: string;

  userId?: string;
  avatarId?: string;
  cartId?: string;
  avatarName?: string;

  paid: boolean;
  createdAt?: string;

  inventoryId: string;

  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productName?: string;
  tokenName?: string;

  listReadableId?: string;

  modelId?: string;

  kind?: string;
  size?: string;
  color?: string;
  rgb?: string | number;
  modelNumber?: string;

  volumeValue?: number;
  volumeUnit?: string;

  categoryId?: string;
  categoryCode?: string;
  categoryNameJa?: string;
  categoryNameEn?: string;
  categoryKind?: string;
  categoryPath?: string[];
  categoryFields?: Record<string, any>;

  qty?: number;
  price?: number;

  transferred: boolean;
  transferredAt?: string;
};

export type OrderListParams =
  PageParams & {
    id?: string;
    userId?: string;
    avatarId?: string;
    cartId?: string;
    createdFrom?: string;
    createdTo?: string;
  };

function buildQuery(
  params: Record<
    string,
    string | number | boolean | undefined
  >,
): string {
  const searchParams =
    new URLSearchParams();

  for (
    const [key, value]
    of Object.entries(params)
  ) {
    if (value === undefined) {
      continue;
    }

    const stringValue =
      String(value);

    if (!stringValue) {
      continue;
    }

    searchParams.set(
      key,
      stringValue,
    );
  }

  const query =
    searchParams.toString();

  return query
    ? `?${query}`
    : "";
}

async function readErrorMessage(
  response: Response,
): Promise<string> {
  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  try {
    if (
      contentType.includes(
        "application/json",
      )
    ) {
      const body =
        await response.json();

      if (
        body &&
        typeof body === "object" &&
        "error" in body
      ) {
        return String(
          body.error,
        );
      }

      return `${response.status} ${response.statusText}`;
    }

    const text =
      await response.text();

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
  const authHeaders =
    await getAuthHeaders();

  const headers =
    new Headers(
      init?.headers ?? {},
    );

  headers.set(
    "Accept",
    "application/json",
  );

  if (
    !headers.has(
      "Content-Type",
    )
  ) {
    headers.set(
      "Content-Type",
      "application/json",
    );
  }

  for (
    const [key, value]
    of Object.entries(
      authHeaders,
    )
  ) {
    if (!headers.has(key)) {
      headers.set(
        key,
        value,
      );
    }
  }

  const response =
    await fetch(
      url,
      {
        ...init,
        headers,
      },
    );

  if (!response.ok) {
    const message =
      await readErrorMessage(
        response,
      );

    throw new Error(message);
  }

  const contentType =
    response.headers.get(
      "content-type",
    ) ?? "";

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    const text =
      await response
        .text()
        .catch(() => "");

    throw new Error(
      "API returned non-JSON response. " +
        `url=${url} ` +
        `content-type=${contentType}` +
        (
          text
            ? ` body=${text.slice(0, 200)}`
            : ""
        ),
    );
  }

  return await response.json() as T;
}

export interface OrderRepository {
  getById(
    id: string,
  ): Promise<Order>;

  listItemInventoryRows(
    params?: OrderListParams,
  ): Promise<
    PageResult<OrderItemInventoryRowDTO>
  >;
}

/**
 * OrderRepositoryを生成する。
 *
 * - URL構築はAPI_BASEを使用する
 * - 通信には標準のfetchを使用する
 * - 認証ヘッダー付与はrequestJSONへ集約する
 */
export function createOrderRepository(): OrderRepository {
  const resolvedBaseUrl =
    API_BASE.replace(
      /\/+$/g,
      "",
    );

  const buildUrl = (
    path: string,
  ): string => {
    const normalizedPath =
      path.replace(
        /^\/+/g,
        "",
      );

    return (
      `${resolvedBaseUrl}/` +
      normalizedPath
    );
  };

  return {
    async getById(
      id: string,
    ): Promise<Order> {
      if (!id) {
        throw new Error(
          "id is required",
        );
      }

      const url =
        buildUrl(
          `/orders/${encodeURIComponent(id)}`,
        );

      return requestJSON<Order>(
        url,
        {
          method: "GET",
        },
      );
    },

    async listItemInventoryRows(
      params: OrderListParams = {},
    ): Promise<
      PageResult<OrderItemInventoryRowDTO>
    > {
      const query =
        buildQuery({
          page:
            params.page ?? 1,

          perPage:
            params.perPage ?? 20,

          id:
            params.id,

          userId:
            params.userId,

          avatarId:
            params.avatarId,

          cartId:
            params.cartId,

          createdFrom:
            params.createdFrom,

          createdTo:
            params.createdTo,
        });

      const url =
        buildUrl(
          `/orders/items${query}`,
        );

      return requestJSON<
        PageResult<OrderItemInventoryRowDTO>
      >(
        url,
        {
          method: "GET",
        },
      );
    },
  };
}