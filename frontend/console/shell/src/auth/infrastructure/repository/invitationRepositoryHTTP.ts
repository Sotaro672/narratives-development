// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts
import { auth } from "../config/firebaseClient";
import { buildConsoleUrl } from "../../../shared/http/apiBase";

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  memberId: string;
  companyId: string;
  companyName?: string;
  assignedBrandIds: string[];
  brandNames?: string[];
  permissions: string[];
  email?: string;
};

export type ValidateResponse = {
  email: string;
  memberId?: string;
  companyId?: string;
  companyName?: string;
  assignedBrandIds?: string[];
  brandNames?: string[];
  permissions?: string[];
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

function normalizeInvitationInfo(data: InvitationInfo): InvitationInfo {
  const companyId = safeTrim(data.companyId);
  const companyName = safeTrim(data.companyName);

  const assignedBrandIds = Array.isArray(data.assignedBrandIds)
    ? data.assignedBrandIds
        .map((id) => safeTrim(id))
        .filter((id) => id.length > 0)
    : [];

  const brandNames = Array.isArray(data.brandNames)
    ? data.brandNames
        .map((name) => safeTrim(name))
        .filter((name) => name.length > 0)
    : [];

  return {
    memberId: safeTrim(data.memberId),
    companyId,
    companyName: companyName || companyId,
    assignedBrandIds,
    brandNames: brandNames.length > 0 ? brandNames : assignedBrandIds,
    permissions: Array.isArray(data.permissions)
      ? data.permissions
          .map((permission) => safeTrim(permission))
          .filter((permission) => permission.length > 0)
      : [],
    email: safeTrim(data.email) || undefined,
  };
}

function normalizeValidateResponse(data: ValidateResponse): ValidateResponse {
  const companyId = safeTrim(data.companyId);
  const companyName = safeTrim(data.companyName);

  const assignedBrandIds = Array.isArray(data.assignedBrandIds)
    ? data.assignedBrandIds
        .map((id) => safeTrim(id))
        .filter((id) => id.length > 0)
    : [];

  const brandNames = Array.isArray(data.brandNames)
    ? data.brandNames
        .map((name) => safeTrim(name))
        .filter((name) => name.length > 0)
    : [];

  return {
    email: safeTrim(data.email),
    memberId: safeTrim(data.memberId) || undefined,
    companyId: companyId || undefined,
    companyName: companyName || companyId || undefined,
    assignedBrandIds: assignedBrandIds.length > 0 ? assignedBrandIds : undefined,
    brandNames:
      brandNames.length > 0
        ? brandNames
        : assignedBrandIds.length > 0
          ? assignedBrandIds
          : undefined,
    permissions: Array.isArray(data.permissions)
      ? data.permissions
          .map((permission) => safeTrim(permission))
          .filter((permission) => permission.length > 0)
      : undefined,
  };
}

function validateResponseToInvitationInfo(
  data: ValidateResponse,
): InvitationInfo {
  const normalized = normalizeValidateResponse(data);

  return normalizeInvitationInfo({
    memberId: normalized.memberId ?? "",
    companyId: normalized.companyId ?? "",
    companyName: normalized.companyName,
    assignedBrandIds: normalized.assignedBrandIds ?? [],
    brandNames: normalized.brandNames,
    permissions: normalized.permissions ?? [],
    email: normalized.email || undefined,
  });
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
// ------------------------------

export async function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  const trimmedToken = token.trim();

  if (!trimmedToken) {
    throw new Error("token が指定されていません。");
  }

  const url = buildConsoleUrl("/invitations/validate");

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      token: trimmedToken,
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