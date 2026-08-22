// frontend/amol/src/features/order/api/orderDetailApi.ts

import { readJsonResponse } from "../../../lib/apiResponse";

import type {
  WalletOrder,
} from "../../shared/types/orderTypes";

export type FetchOrderDetailInput = {
  backendUrl: string;
  idToken: string;
  orderId: string;
};

export async function fetchOrderDetail({
  backendUrl,
  idToken,
  orderId,
}: FetchOrderDetailInput): Promise<WalletOrder> {
  const normalizedOrderId =
    orderId.trim();

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

  return readJsonResponse<WalletOrder>(
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