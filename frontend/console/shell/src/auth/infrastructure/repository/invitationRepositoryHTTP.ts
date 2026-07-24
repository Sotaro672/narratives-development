// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts
import { auth } from "../config/firebaseClient";
import { buildConsoleUrl } from "../../../shared/http/apiBase";

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  companyName?: string;
  brandNames?: string[];
};

export type ValidateResponse = {
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
};

// ------------------------------
// Helpers
// ------------------------------

function safeTrim(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

async function getFirebaseIdToken(): Promise<string> {
  const currentUser = auth.currentUser;

  if (!currentUser) {
    throw new Error(
      "Firebase Authenticationへのサインインが確認できません。",
    );
  }

  const idToken = await currentUser.getIdToken();

  if (!idToken) {
    throw new Error("Firebase ID tokenを取得できませんでした。");
  }

  return idToken;
}

function normalizeCompleteInvitationPayload(
  payload: CompleteInvitationBackendPayload,
): CompleteInvitationBackendPayload {
  const normalized: CompleteInvitationBackendPayload = {
    token: safeTrim(payload.token),
    lastName: safeTrim(payload.lastName),
    lastNameKana: safeTrim(payload.lastNameKana),
    firstName: safeTrim(payload.firstName),
    firstNameKana: safeTrim(payload.firstNameKana),
  };

  if (!normalized.token) {
    throw new Error("token が指定されていません。");
  }

  if (!normalized.lastName) {
    throw new Error("lastName が指定されていません。");
  }

  if (!normalized.lastNameKana) {
    throw new Error("lastNameKana が指定されていません。");
  }

  if (!normalized.firstName) {
    throw new Error("firstName が指定されていません。");
  }

  if (!normalized.firstNameKana) {
    throw new Error("firstNameKana が指定されていません。");
  }

  return normalized;
}

function normalizeStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  const normalized = value
    .map((item) => safeTrim(item))
    .filter((item) => item.length > 0);

  return Array.from(new Set(normalized));
}

function normalizeValidateResponse(
  data: ValidateResponse,
): ValidateResponse {
  const companyName = safeTrim(data.companyName);
  const brandNames = normalizeStringArray(data.brandNames);

  return {
    companyName: companyName || undefined,
    brandNames: brandNames.length > 0 ? brandNames : undefined,
  };
}

function validateResponseToInvitationInfo(
  data: ValidateResponse,
): InvitationInfo {
  const normalized = normalizeValidateResponse(data);

  return {
    companyName: normalized.companyName,
    brandNames: normalized.brandNames,
  };
}

async function parseErrorMessage(
  res: Response,
  text: string,
  fallback: string,
): Promise<string> {
  let message = `${fallback} (status ${res.status})`;

  try {
    const errorResponse = JSON.parse(text) as ErrorResponse;

    if (errorResponse.error) {
      message = errorResponse.error;
    }
  } catch {
    // JSON形式でないエラーレスポンスはfallbackを使用する。
  }

  return message;
}

// ------------------------------
// 招待情報取得
// - POST /invitations/validate
// ------------------------------

export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  const data = await validateInvitation(token);

  return validateResponseToInvitationInfo(data);
}

// ------------------------------
// validateInvitation
// - POST /invitations/validate
// - 招待受諾前の公開APIのためAuthorizationは付与しない
// - 会社名とブランド名以外の機微情報は取得しない
// ------------------------------

export async function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  const normalizedToken = safeTrim(token);

  if (!normalizedToken) {
    throw new Error("token が指定されていません。");
  }

  const url = buildConsoleUrl("/invitations/validate");

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      token: normalizedToken,
    }),
  });

  const text = await res.text();

  if (!res.ok) {
    const message = await parseErrorMessage(
      res,
      text,
      "招待の検証に失敗しました",
    );

    throw new Error(message);
  }

  if (!text) {
    return {};
  }

  const data = JSON.parse(text) as ValidateResponse;

  return normalizeValidateResponse(data);
}

// ------------------------------
// completeInvitationOnBackend
// - POST /invitations/complete
// - UIDとemailはbodyへ送信しない
// - Firebase ID tokenをAuthorizationへ付与する
// - Backend側でID tokenからUIDとemailを取得する
// ------------------------------

export async function completeInvitationOnBackend(
  payload: CompleteInvitationBackendPayload,
): Promise<void> {
  const url = buildConsoleUrl("/invitations/complete");
  const body = normalizeCompleteInvitationPayload(payload);
  const idToken = await getFirebaseIdToken();

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await res.text();

  if (!res.ok) {
    const message = await parseErrorMessage(
      res,
      text,
      "招待の完了処理に失敗しました",
    );

    throw new Error(message);
  }
}