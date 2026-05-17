// frontend/console/order/src/application/orderDetailBuilder.ts
import {
  Order,
  OrderItemInventoryRowDTO,
} from "../infrastructure/repostiroty";

export type OrderDetailItemDTO = {
  size?: string;
  color?: string;
  rgb?: string;
  modelNumber?: string;

  kind?: string;
  volumeValue?: number;
  volumeUnit?: string;

  productName?: string;
  tokenName?: string;

  listId?: string;

  qty?: number;
  price?: number;
  transferred: boolean;
  transferredAt?: string;

  categoryId?: string;
  categoryCode?: string;
  categoryNameJa?: string;
  categoryNameEn?: string;
  categoryKind?: string;
  categoryPath?: string[];
  categoryFields?: Record<string, any>;

  [k: string]: any;
};

export type OrderDetailDTO = {
  id: string;

  userName?: string;
  avatarName?: string;

  cartId?: string;
  paid: boolean;
  createdAt?: string;

  shippingSnapshot?: {
    zipCode?: string;
    state?: string;
    city?: string;
    street?: string;
    street2?: string;
    country?: string;
    [k: string]: any;
  };

  billingSnapshot?: {
    [k: string]: any;
  };

  items?: OrderDetailItemDTO[];
};

function toNumberOrUndefined(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string" && value.trim() !== "") {
    const n = Number(value);
    if (Number.isFinite(n)) {
      return n;
    }
  }

  return undefined;
}

function toNumberOrZero(value: unknown): number {
  const n = toNumberOrUndefined(value);
  return n ?? 0;
}

function toStringArray(value: unknown): string[] | undefined {
  if (!Array.isArray(value)) {
    return undefined;
  }

  const out = value
    .map((v) => String(v ?? "").trim())
    .filter((v) => v !== "");

  return out.length > 0 ? out : undefined;
}

function toRecord(value: unknown): Record<string, any> | undefined {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return undefined;
  }

  const out: Record<string, any> = {};

  for (const [key, v] of Object.entries(value as Record<string, any>)) {
    if (!key) continue;
    out[key] = v;
  }

  return Object.keys(out).length > 0 ? out : undefined;
}

// Order(= /orders/{id}) をベースに、/orders/items の items だけで items を作り直す
export function buildOrderDetailFromAllowedItems(
  base: Order,
  allowedRows: OrderItemInventoryRowDTO[],
): OrderDetailDTO {
  const orderId = String((base as any).id ?? "");

  const byOrder = allowedRows.filter(
    (r) => String((r as any).orderId ?? "") === orderId,
  );

  const items: OrderDetailItemDTO[] = byOrder.map((r) => {
    const volumeValue = toNumberOrUndefined((r as any).volumeValue);

    return {
      size: String((r as any).size ?? ""),
      color: String((r as any).color ?? ""),
      rgb: String((r as any).rgb ?? ""),
      modelNumber: String((r as any).modelNumber ?? ""),

      kind: String((r as any).kind ?? ""),
      volumeValue,
      volumeUnit: String((r as any).volumeUnit ?? ""),

      productName: String((r as any).productName ?? ""),
      tokenName: String((r as any).tokenName ?? ""),

      listId: String((r as any).listReadableId ?? ""),

      qty: toNumberOrZero((r as any).qty),
      price: toNumberOrZero((r as any).price),

      transferred: Boolean((r as any).transferred),
      transferredAt: String((r as any).transferredAt ?? ""),

      categoryId: String((r as any).categoryId ?? ""),
      categoryCode: String((r as any).categoryCode ?? ""),
      categoryNameJa: String((r as any).categoryNameJa ?? ""),
      categoryNameEn: String((r as any).categoryNameEn ?? ""),
      categoryKind: String((r as any).categoryKind ?? ""),
      categoryPath: toStringArray((r as any).categoryPath),
      categoryFields: toRecord((r as any).categoryFields),
    };
  });

  const firstRow = byOrder[0] as any;

  return {
    id: orderId,

    userName: String((base as any).userName ?? ""),
    avatarName: String(firstRow?.avatarName ?? (base as any).avatarName ?? ""),

    cartId: String((base as any).cartId ?? ""),
    paid: Boolean((base as any).paid),
    createdAt: String((base as any).createdAt ?? ""),

    shippingSnapshot: {
      zipCode: String((base as any)?.shippingSnapshot?.zipCode ?? ""),
      state: String((base as any)?.shippingSnapshot?.state ?? ""),
      city: String((base as any)?.shippingSnapshot?.city ?? ""),
      street: String((base as any)?.shippingSnapshot?.street ?? ""),
      street2: String((base as any)?.shippingSnapshot?.street2 ?? ""),
      country: String((base as any)?.shippingSnapshot?.country ?? ""),
    },

    billingSnapshot: (base as any).billingSnapshot,
    items,
  };
}