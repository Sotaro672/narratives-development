// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts
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
  uid: string;
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
  email: string;
};

export type LegacyCompleteInvitationBackendPayload = {
  token: string;
  uid: string;
  email?: string;
  profile?: {
    lastName?: string;
    lastNameKana?: string;
    firstName?: string;
    firstNameKana?: string;
  };
};

type CompleteInvitationPayloadInput =
  | CompleteInvitationBackendPayload
  | LegacyCompleteInvitationBackendPayload;

type ErrorResponse = {
  error?: string;
};

// ------------------------------
// Helpers
// ------------------------------

function safeTrim(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}

function normalizeCompleteInvitationPayload(
  payload: CompleteInvitationPayloadInput,
): CompleteInvitationBackendPayload {
  const lastName =
    "lastName" in payload
      ? safeTrim(payload.lastName)
      : safeTrim(payload.profile?.lastName);

  const lastNameKana =
    "lastNameKana" in payload
      ? safeTrim(payload.lastNameKana)
      : safeTrim(payload.profile?.lastNameKana);

  const firstName =
    "firstName" in payload
      ? safeTrim(payload.firstName)
      : safeTrim(payload.profile?.firstName);

  const firstNameKana =
    "firstNameKana" in payload
      ? safeTrim(payload.firstNameKana)
      : safeTrim(payload.profile?.firstNameKana);

  const normalized: CompleteInvitationBackendPayload = {
    token: safeTrim(payload.token),
    uid: safeTrim(payload.uid),
    lastName,
    lastNameKana,
    firstName,
    firstNameKana,
    email: safeTrim(payload.email),
  };

  if (!normalized.token) {
    throw new Error("token が指定されていません。");
  }
  if (!normalized.uid) {
    throw new Error("uid が指定されていません。");
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
  if (!normalized.email) {
    throw new Error("email が指定されていません。");
  }

  return normalized;
}

function normalizeInvitationInfo(data: InvitationInfo): InvitationInfo {
  const companyId = safeTrim(data.companyId);
  const companyName = safeTrim(data.companyName);

  const assignedBrandIds = Array.isArray(data.assignedBrandIds)
    ? data.assignedBrandIds.map((id) => safeTrim(id)).filter((id) => id.length > 0)
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
      ? data.permissions.map((p) => safeTrim(p)).filter((p) => p.length > 0)
      : [],
    email: safeTrim(data.email) || undefined,
  };
}

function normalizeValidateResponse(data: ValidateResponse): ValidateResponse {
  const companyId = safeTrim(data.companyId);
  const companyName = safeTrim(data.companyName);

  const assignedBrandIds = Array.isArray(data.assignedBrandIds)
    ? data.assignedBrandIds.map((id) => safeTrim(id)).filter((id) => id.length > 0)
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
    brandNames: brandNames.length > 0 ? brandNames : assignedBrandIds,
    permissions: Array.isArray(data.permissions)
      ? data.permissions.map((p) => safeTrim(p)).filter((p) => p.length > 0)
      : undefined,
  };
}

function validateResponseToInvitationInfo(data: ValidateResponse): InvitationInfo {
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
  let msg = `${fallback} (status ${res.status})`;

  try {
    const errJson = JSON.parse(text) as ErrorResponse;
    if (errJson.error) {
      msg = errJson.error;
    }
  } catch {
    // ignore
  }

  return msg;
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
// ------------------------------

export async function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  const trimmed = token.trim();
  if (!trimmed) {
    throw new Error("token が指定されていません。");
  }

  const url = buildConsoleUrl("/invitations/validate");

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token: trimmed }),
  });

  const text = await res.text();

  if (!res.ok) {
    const msg = await parseErrorMessage(
      res,
      text,
      "招待の検証に失敗しました",
    );
    throw new Error(msg);
  }

  const data = JSON.parse(text) as ValidateResponse;
  return normalizeValidateResponse(data);
}

// ------------------------------
// completeInvitationOnBackend
// - POST /invitations/complete
// ------------------------------

export async function completeInvitationOnBackend(
  payload: CompleteInvitationPayloadInput,
): Promise<void> {
  const url = buildConsoleUrl("/invitations/complete");

  const body = normalizeCompleteInvitationPayload(payload);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });

  const text = await res.text();

  if (!res.ok) {
    const msg = await parseErrorMessage(
      res,
      text,
      "招待の完了処理に失敗しました",
    );
    throw new Error(msg);
  }
}