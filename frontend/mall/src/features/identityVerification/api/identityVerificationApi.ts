// frontend/mall/src/features/identityVerification/api/identityVerificationApi.ts

import { requestJson } from "../../../lib/http";
import type { IdentityVerification } from "../types";

const IDENTITY_VERIFICATION_BASE_PATH =
  "/mall/me/identity-verification";

export type IdentityVerificationRequestOptions = {
  signal?: AbortSignal;
};

async function requestIdentityVerification<T>(
  path: string,
  init?: Omit<RequestInit, "body">,
): Promise<T> {
  return requestJson<T>(path, {
    ...init,
    auth: "required",
    messages: {
      requestErrorMessage:
        "本人確認情報のAPIリクエストに失敗しました。",
    },
  });
}

export async function fetchIdentityVerification(
  options: IdentityVerificationRequestOptions = {},
): Promise<IdentityVerification> {
  return requestIdentityVerification<IdentityVerification>(
    IDENTITY_VERIFICATION_BASE_PATH,
    {
      method: "GET",
      signal: options.signal,
    },
  );
}

export async function verifyIdentityWithMockMyNumberCard(
  options: IdentityVerificationRequestOptions = {},
): Promise<IdentityVerification> {
  return requestIdentityVerification<IdentityVerification>(
    `${IDENTITY_VERIFICATION_BASE_PATH}/mock`,
    {
      method: "POST",
      signal: options.signal,
    },
  );
}