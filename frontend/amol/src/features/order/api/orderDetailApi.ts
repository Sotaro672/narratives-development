// frontend/amol/src/features/order/api/orderDetailApi.ts

import { readJsonResponse } from "../../../lib/apiResponse";

import type {
  FetchOrderDetailInput,
  OrderDetail,
} from "../../shared/types/orderDetailTypes";

export type CancelOrderItemInput =
  FetchOrderDetailInput & {
    itemIndex: number;
  };

export async function fetchOrderDetail({
  backendUrl,
  idToken,
  orderId,
}: FetchOrderDetailInput): Promise<OrderDetail> {
  const normalizedOrderId = orderId.trim();

  if (!normalizedOrderId) {
    throw new Error(
      "注文IDが指定されていません。",
    );
  }

  const response = await fetch(
    `${backendUrl}/mall/me/orders/${encodeURIComponent(
      normalizedOrderId,
    )}`,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  return readJsonResponse<OrderDetail>(
    response,
    {
      requestErrorMessage:
        "注文情報の取得に失敗しました。",
      nonJsonErrorMessage:
        "注文詳細APIがJSON以外を返しました。",
      invalidJsonErrorMessage:
        "注文詳細APIのJSON形式が不正です。",
    },
  );
}

export async function cancelOrderItem({
  backendUrl,
  idToken,
  orderId,
  itemIndex,
}: CancelOrderItemInput): Promise<OrderDetail> {
  const normalizedOrderId = orderId.trim();

  if (!normalizedOrderId) {
    throw new Error(
      "注文IDが指定されていません。",
    );
  }

  if (
    !Number.isInteger(itemIndex) ||
    itemIndex < 0
  ) {
    throw new Error(
      "注文商品のインデックスが不正です。",
    );
  }

  const response = await fetch(
    `${backendUrl}/mall/me/orders/${encodeURIComponent(
      normalizedOrderId,
    )}/items/${itemIndex}/cancel`,
    {
      method: "PATCH",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  return readJsonResponse<OrderDetail>(
    response,
    {
      requestErrorMessage:
        "商品のキャンセルに失敗しました。",
      nonJsonErrorMessage:
        "商品キャンセルAPIがJSON以外を返しました。",
      invalidJsonErrorMessage:
        "商品キャンセルAPIのJSON形式が不正です。",
    },
  );
}