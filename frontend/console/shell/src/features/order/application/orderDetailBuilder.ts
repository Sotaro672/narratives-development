// frontend/console/shell/src/features/order/application/orderDetailBuilder.ts

import type {
  Order,
  OrderItemInventoryRowDTO,
  ShippingSnapshot,
} from "../infrastructure/repository";

export type OrderDetailItemDTO = {
  size?: string;
  color?: string;
  rgb?: string | number;
  modelNumber?: string;

  kind?: string;
  volumeValue?: number;
  volumeUnit?: string;

  productName?: string;
  tokenName?: string;

  listId?: string;

  qty: number;
  price: number;

  transferred: boolean;
  transferredAt?: string;

  categoryId?: string;
  categoryCode?: string;
  categoryNameJa?: string;
  categoryNameEn?: string;
  categoryKind?: string;
  categoryPath?: string[];
  categoryFields?: Record<string, unknown>;
};

export type OrderDetailDTO = {
  id: string;

  userName?: string;
  avatarName?: string;

  cartId?: string;
  paid: boolean;
  createdAt?: string;

  shippingSnapshot?: ShippingSnapshot;

  items: OrderDetailItemDTO[];
};

/**
 * /orders/{id}の注文情報と、
 * /orders/itemsのアイテム情報から注文詳細DTOを生成する。
 *
 * allowedRowsは呼び出し元で注文IDを指定して取得済みのため、
 * frontendでは注文IDによる再絞り込みを行わない。
 */
export function buildOrderDetailFromAllowedItems(
  base: Order,
  allowedRows: OrderItemInventoryRowDTO[],
): OrderDetailDTO {
  const items: OrderDetailItemDTO[] =
    allowedRows.map(
      (
        row,
      ): OrderDetailItemDTO => ({
        size:
          row.size,

        color:
          row.color,

        rgb:
          row.rgb,

        modelNumber:
          row.modelNumber,

        kind:
          row.kind,

        volumeValue:
          row.volumeValue,

        volumeUnit:
          row.volumeUnit,

        productName:
          row.productName,

        tokenName:
          row.tokenName,

        listId:
          row.listReadableId,

        qty:
          row.qty ?? 0,

        price:
          row.price ?? 0,

        transferred:
          row.transferred,

        transferredAt:
          row.transferredAt,

        categoryId:
          row.categoryId,

        categoryCode:
          row.categoryCode,

        categoryNameJa:
          row.categoryNameJa,

        categoryNameEn:
          row.categoryNameEn,

        categoryKind:
          row.categoryKind,

        categoryPath:
          row.categoryPath,

        categoryFields:
          row.categoryFields,
      }),
    );

  return {
    id:
      base.id,

    userName:
      base.userName,

    avatarName:
      allowedRows[0]?.avatarName ??
      base.avatarName,

    cartId:
      base.cartId,

    paid:
      base.paid,

    createdAt:
      base.createdAt,

    shippingSnapshot:
      base.shippingSnapshot,

    items,
  };
}