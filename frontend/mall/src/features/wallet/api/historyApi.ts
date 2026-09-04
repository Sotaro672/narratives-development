// frontend/amol/src/features/wallet/api/historyApi.ts

import { readJsonResponse } from "../../../lib/apiResponse";

import type {
  FetchWalletOrdersInput,
  WalletOrdersPage,
} from "../../shared/types/orderTypes";

export async function fetchWalletOrders({
  backendUrl,
  idToken,
  page = 1,
  perPage = 20,
  sort = "createdAt",
  order = "desc",
}: FetchWalletOrdersInput): Promise<WalletOrdersPage> {
  const url = new URL(`${backendUrl}/mall/me/orders`);

  url.searchParams.set("page", String(page));
  url.searchParams.set("perPage", String(perPage));
  url.searchParams.set("sort", sort);
  url.searchParams.set("order", order);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
  });

  return readJsonResponse<WalletOrdersPage>(response, {
    requestErrorMessage: "注文履歴の取得に失敗しました。",
    nonJsonErrorMessage: "注文履歴APIがJSON以外を返しました。",
    invalidJsonErrorMessage: "注文履歴APIのJSON形式が不正です。",
  });
}