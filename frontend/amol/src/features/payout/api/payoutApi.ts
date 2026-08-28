// frontend/amol/src/features/payout/api/payoutApi.ts

import { buildBackendUrl } from "../../../lib/apiBaseUrl";

import type {
  PayoutAccount,
  PayoutAccountResponse,
  PayoutAccountSessionResponse,
} from "../../shared/types/payoutAccount";

type AuthenticatedRequestInput = {
  idToken: string;
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

export async function createPayoutAccountSession({
  idToken,
}: AuthenticatedRequestInput): Promise<string> {
  const response = await fetch(
    buildBackendUrl(
      "/mall/me/payout-account/account-session"
    ),
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
        Accept: "application/json",
      },
    }
  );

  const body =
    await readJson<PayoutAccountSessionResponse>(
      response
    );

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "Stripeとの接続に失敗しました。"
    );
  }

  const clientSecret = body?.data?.clientSecret;

  if (!clientSecret) {
    throw new Error(
      "StripeのAccount Sessionを取得できませんでした。"
    );
  }

  return clientSecret;
}