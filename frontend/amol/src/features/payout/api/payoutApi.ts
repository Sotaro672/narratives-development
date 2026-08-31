// frontend/amol/src/features/payout/api/payoutApi.ts

import { buildBackendUrl } from "../../../lib/apiBaseUrl";

import type {
  PayoutAccount,
  PayoutAccountRegistrationInput,
  PayoutAccountResponse,
} from "../../shared/types/payoutAccount";

type AuthenticatedRequestInput = {
  idToken: string;
};

type RegisterPayoutAccountRequestInput = AuthenticatedRequestInput & {
  input: PayoutAccountRegistrationInput;
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
  const response = await fetch(buildBackendUrl("/mall/me/payout-account"), {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      Accept: "application/json",
    },
  });

  const body = await readJson<PayoutAccountResponse>(response);
  if (!response.ok) {
    throw new Error(body?.error || "売上受取口座の情報取得に失敗しました。");
  }

  return body?.data || null;
}

export async function registerPayoutAccount({
  idToken,
  input,
}: RegisterPayoutAccountRequestInput): Promise<PayoutAccount> {
  const response = await fetch(buildBackendUrl("/mall/me/payout-account"), {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${idToken}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      bankCode: input.bankCode,
      bankName: input.bankName,
      branchCode: input.branchCode,
      branchName: input.branchName,
      accountType: input.accountType,
      accountNumber: input.accountNumber,
      accountHolderName: input.accountHolderName,
    }),
  });

  const body = await readJson<PayoutAccountResponse>(response);
  if (!response.ok) {
    throw new Error(body?.error || "売上受取口座の登録に失敗しました。");
  }

  const payoutAccount = body?.data;
  if (!payoutAccount) {
    throw new Error("登録した売上受取口座の情報を取得できませんでした。");
  }

  return payoutAccount;
}