// frontend/amol/src/features/payout/api/payoutApi.ts

import { buildBackendUrl } from "../../../lib/apiBaseUrl";

import type {
  PayoutAccount,
  PayoutAccountLinkResponse,
  PayoutAccountResponse,
} from "../../shared/types/payoutAccount";

type AuthenticatedRequestInput = {
  idToken: string;
};

export type CreatePayoutAccountLinkInput = AuthenticatedRequestInput & {
  returnUrl: string;
  refreshUrl: string;
};

async function readJson<T>(
  response: Response
): Promise<T | null> {
  const contentType =
    response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    return null;
  }

  return (await response.json()) as T;
}

export async function fetchPayoutAccount({
  idToken,
}: AuthenticatedRequestInput): Promise<PayoutAccount | null> {
  const response = await fetch(
    buildBackendUrl("/mall/me/payout-account"),
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
        Accept: "application/json",
      },
    }
  );

  const body =
    await readJson<PayoutAccountResponse>(response);

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "売上受取口座の情報取得に失敗しました。"
    );
  }

  return body?.data || null;
}

export async function createPayoutAccountLink({
  idToken,
  returnUrl,
  refreshUrl,
}: CreatePayoutAccountLinkInput): Promise<string> {
  const response = await fetch(
    buildBackendUrl(
      "/mall/me/payout-account/account-link"
    ),
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify({
        returnUrl,
        refreshUrl,
      }),
    }
  );

  const body =
    await readJson<PayoutAccountLinkResponse>(
      response
    );

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "Stripeとの接続に失敗しました。"
    );
  }

  const url = body?.data?.url;

  if (!url) {
    throw new Error(
      "Stripeの口座登録URLを取得できませんでした。"
    );
  }

  return url;
}