// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts

import { buildConsoleUrl } from "../../../shared/http/apiBase";
import { fetchJSON, HttpError } from "../../../shared/http/fetchJSON";

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  companyName?: string;
  brandNames?: string[];
};

export type CompleteInvitationBackendPayload = {
  token: string;
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
};

type ErrorResponse = {
  error?: string;
  message?: string;
};

// ------------------------------
// Error handling
// ------------------------------

function parseBackendErrorBody(bodyText: string | undefined): string | null {
  if (!bodyText) {
    return null;
  }

  try {
    const parsed = JSON.parse(bodyText) as ErrorResponse;

    if (typeof parsed.error === "string" && parsed.error) {
      return parsed.error;
    }

    if (typeof parsed.message === "string" && parsed.message) {
      return parsed.message;
    }

    return null;
  } catch {
    return null;
  }
}

function toInvitationRepositoryError(error: unknown, fallback: string): Error {
  if (error instanceof HttpError) {
    const backendMessage = parseBackendErrorBody(error.bodyText);

    if (backendMessage) {
      return new Error(backendMessage);
    }

    return new Error(`${fallback} (status ${error.status})`);
  }

  if (error instanceof Error) {
    return error;
  }

  return new Error(fallback);
}

// ------------------------------
// 招待情報取得
// - POST /invitations/validate
// ------------------------------

export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  return validateInvitation(token);
}

// ------------------------------
// validateInvitation
// - POST /invitations/validate
// - 招待受諾前の公開APIのためAuthorizationは付与しない
// - Backend BFFレスポンスをそのまま正とする
// ------------------------------

export async function validateInvitation(
  token: string,
): Promise<InvitationInfo> {
  const url = buildConsoleUrl("/invitations/validate");

  try {
    return await fetchJSON<InvitationInfo>(url, {
      method: "POST",
      auth: "none",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ token }),
    });
  } catch (error: unknown) {
    throw toInvitationRepositoryError(
      error,
      "招待の検証に失敗しました",
    );
  }
}

// ------------------------------
// completeInvitationOnBackend
// - POST /invitations/complete
// - UIDとemailはbodyへ送信しない
// - Firebase ID tokenはfetchJSONがAuthorizationへ付与する
// - 401時はID tokenを強制更新して1回だけ再送する
// - Backend側でID tokenからUIDとemailを取得する
// ------------------------------

export async function completeInvitationOnBackend(
  payload: CompleteInvitationBackendPayload,
): Promise<void> {
  const url = buildConsoleUrl("/invitations/complete");

  try {
    await fetchJSON<void>(url, {
      method: "POST",
      auth: "required",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
      allowNonJson: true,
    });
  } catch (error: unknown) {
    throw toInvitationRepositoryError(
      error,
      "招待の完了処理に失敗しました",
    );
  }
}