// frontend/amol/src/features/payout/api/payoutApi.ts

import { buildBackendUrl } from "../../../lib/apiBaseUrl";

import type {
  PayoutAccount,
  PayoutAccountResponse,
  PayoutAccountTokenRegistrationInput,
} from "../../shared/types/payoutAccount";

type AuthenticatedRequestInput = {
  idToken: string;
};

type RegisterPayoutAccountRequestInput = AuthenticatedRequestInput & {
  input: PayoutAccountTokenRegistrationInput;
};

type PayoutAccountSessionResponse = {
  data?: {
    clientSecret?: string;
  } | null;
  error?: string;
};

async function readJson<T>(response: Response): Promise<T | null> {
  const contentType = response.headers.get("content-type") || "";

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
    },
  );

  const body = await readJson<PayoutAccountResponse>(response);

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "売上受取口座の情報取得に失敗しました。",
    );
  }

  return body?.data || null;
}

// Deprecated.
//
// 独自の口座登録画面へ戻したため、新しい登録フローでは使用しない。
// PayoutAccountOnboardingPage.tsx を削除するまでのコンパイル互換性として残す。
export async function createPayoutAccountSession({
  idToken,
}: AuthenticatedRequestInput): Promise<string> {
  const response = await fetch(
    buildBackendUrl("/mall/me/payout-account/session"),
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${idToken}`,
        Accept: "application/json",
      },
    },
  );

  const body = await readJson<PayoutAccountSessionResponse>(response);

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "売上受取口座の登録セッション作成に失敗しました。",
    );
  }

  const clientSecret = body?.data?.clientSecret?.trim() || "";

  if (!clientSecret) {
    throw new Error(
      "売上受取口座の登録セッションを取得できませんでした。",
    );
  }

  return clientSecret;
}

export async function registerPayoutAccount({
  idToken,
  input,
}: RegisterPayoutAccountRequestInput): Promise<PayoutAccount> {
  const bankAccountToken = input.bankAccountToken.trim();

  if (!bankAccountToken) {
    throw new Error(
      "Stripeの銀行口座トークンを確認できませんでした。",
    );
  }

  const response = await fetch(
    buildBackendUrl("/mall/me/payout-account"),
    {
      method: "PUT",
      headers: {
        Authorization: `Bearer ${idToken}`,
        Accept: "application/json",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        bankAccountToken,
      }),
    },
  );

  const body = await readJson<PayoutAccountResponse>(response);

  if (!response.ok) {
    throw new Error(
      body?.error ||
        "売上受取口座の登録に失敗しました。",
    );
  }

  const payoutAccount = body?.data;

  if (!payoutAccount) {
    throw new Error(
      "登録した売上受取口座の情報を取得できませんでした。",
    );
  }

  return payoutAccount;
}