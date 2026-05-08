//frontend\console\shell\src\auth\infrastructure\repository\invitationRepositoryHTTP.ts
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

export type CompanyResponse = {
  id: string;
  name?: string;
};

export type BrandResponse = {
  id: string;
  name?: string;
};

// 新形式
export type CompleteInvitationBackendPayload = {
  token: string;
  uid: string;
  lastName: string;
  lastNameKana: string;
  firstName: string;
  firstNameKana: string;
  email: string;
};

// 旧形式も受けられるようにしておく
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
    companyName: (companyName || companyId) || undefined,
    assignedBrandIds: assignedBrandIds.length > 0 ? assignedBrandIds : undefined,
    brandNames: brandNames.length > 0 ? brandNames : assignedBrandIds,
    permissions: Array.isArray(data.permissions)
      ? data.permissions.map((p) => safeTrim(p)).filter((p) => p.length > 0)
      : undefined,
  };
}

// ------------------------------
// 招待情報取得（GET /api/invitation）
// ------------------------------
export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  const trimmed = token.trim();
  if (!trimmed) {
    throw new Error("token が指定されていません。");
  }

  const url = buildConsoleUrl(
    `/api/invitation?token=${encodeURIComponent(trimmed)}`,
  );

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  const text = await res.text();

  if (!res.ok) {
    let msg = `Failed to load invitation info (status ${res.status})`;
    try {
      const errJson = JSON.parse(text) as ErrorResponse;
      if (errJson.error) {
        msg = errJson.error;
      }
    } catch {
      // ignore
    }
    throw new Error(msg);
  }

  const data = JSON.parse(text) as InvitationInfo;
  return normalizeInvitationInfo(data);
}

// ------------------------------
// companyId → companyName 取得ヘルパ
// ※ 招待ページは未ログインの可能性があるため、失敗時は ID を返す
// ------------------------------
export async function fetchCompanyNameById(companyId: string): Promise<string> {
  const trimmed = companyId.trim();
  if (!trimmed) {
    return "";
  }

  const url = buildConsoleUrl(`/companies/${encodeURIComponent(trimmed)}`);

  try {
    const res = await fetch(url, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    const text = await res.text();

    if (!res.ok) {
      return trimmed;
    }

    const data = JSON.parse(text) as CompanyResponse;
    const name = (data.name ?? "").trim();
    return name || trimmed;
  } catch {
    return trimmed;
  }
}

// ------------------------------
// assignedBrandId(s) → brandName(s) 取得ヘルパ
// ※ 招待ページは未ログインの可能性があるため、失敗時は ID を返す
// ------------------------------
export async function fetchBrandNameById(brandId: string): Promise<string> {
  const trimmed = brandId.trim();
  if (!trimmed) {
    return "";
  }

  const url = buildConsoleUrl(`/brands/${encodeURIComponent(trimmed)}`);

  try {
    const res = await fetch(url, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
    });

    const text = await res.text();

    if (!res.ok) {
      return trimmed;
    }

    const data = JSON.parse(text) as BrandResponse;
    const name = (data.name ?? "").trim();
    return name || trimmed;
  } catch {
    return trimmed;
  }
}

// assignedBrandIds 全体を brandName[] に変換するヘルパ
export async function fetchBrandNamesByIds(
  assignedBrandIds: string[],
): Promise<string[]> {
  const ids = assignedBrandIds
    .map((id) => id.trim())
    .filter((id) => id.length > 0);

  if (ids.length === 0) return [];

  const tasks = ids.map(async (id) => {
    try {
      return await fetchBrandNameById(id);
    } catch {
      return id;
    }
  });

  return Promise.all(tasks);
}

// ------------------------------
// validateInvitation (POST /api/invitation/validate)
// ------------------------------
export async function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  const trimmed = token.trim();
  if (!trimmed) {
    throw new Error("token が指定されていません。");
  }

  const url = buildConsoleUrl("/api/invitation/validate");

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token: trimmed }),
  });

  const text = await res.text();

  if (!res.ok) {
    let msg = `招待の検証に失敗しました (status ${res.status})`;
    try {
      const errJson = JSON.parse(text) as ErrorResponse;
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }

  const data = JSON.parse(text) as ValidateResponse;
  return normalizeValidateResponse(data);
}

// ------------------------------
// completeInvitationOnBackend (POST /api/invitation/complete)
// ------------------------------
export async function completeInvitationOnBackend(
  payload: CompleteInvitationPayloadInput,
): Promise<void> {
  const url = buildConsoleUrl("/api/invitation/complete");

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
    let msg = `招待の完了処理に失敗しました (status ${res.status})`;
    try {
      const errJson = JSON.parse(text) as ErrorResponse;
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }
}