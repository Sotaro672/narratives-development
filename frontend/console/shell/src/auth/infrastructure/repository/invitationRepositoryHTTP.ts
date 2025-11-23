// frontend/console/shell/src/auth/infrastructure/repository/invitationRepositoryHTTP.ts

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = ENV_BASE || FALLBACK_BASE;

// ------------------------------
// 型定義
// ------------------------------

export type InvitationInfo = {
  memberId: string;
  companyId: string;
  assignedBrandIds: string[];
  permissions: string[];
  email?: string; // Firestore の email を optional で受ける
};

export type ValidateResponse = {
  email: string;
  memberId?: string;
  companyId?: string;
  assignedBrandIds?: string[];
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

export type CompleteInvitationBackendPayload = {
  token: string;
  uid: string;
  profile: {
    lastName: string;
    lastNameKana: string;
    firstName: string;
    firstNameKana: string;
  };
  companyId: string;
  assignedBrandIds: string[];
  permissions: string[];
};

// ------------------------------
// 招待情報取得（GET /api/invitation）
// ------------------------------
export async function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  const url = `${API_BASE}/api/invitation?token=${encodeURIComponent(token)}`;

  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] Fetching invitation info:", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] raw response:", text);

  if (!res.ok) {
    throw new Error(`Failed to load invitation info (status ${res.status})`);
  }

  const data = JSON.parse(text) as InvitationInfo;
  return data;
}

// ------------------------------
// companyId → companyName 取得ヘルパ
// ------------------------------
export async function fetchCompanyNameById(companyId: string): Promise<string> {
  const trimmed = companyId.trim();
  if (!trimmed) {
    throw new Error("companyId が指定されていません。");
  }

  const url = `${API_BASE}/companies/${encodeURIComponent(trimmed)}`;
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] Fetching company name:", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] company response:", text);

  if (!res.ok) {
    throw new Error(`会社情報の取得に失敗しました (status ${res.status})`);
  }

  const data = JSON.parse(text) as CompanyResponse;
  const name = (data.name ?? "").trim();
  if (!name) {
    throw new Error("会社名が取得できませんでした。");
  }
  return name;
}

// ------------------------------
// assignedBrandId(s) → brandName(s) 取得ヘルパ
// ------------------------------
export async function fetchBrandNameById(brandId: string): Promise<string> {
  const trimmed = brandId.trim();
  if (!trimmed) {
    throw new Error("brandId が指定されていません。");
  }

  const url = `${API_BASE}/brands/${encodeURIComponent(trimmed)}`;
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] Fetching brand name:", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
    },
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] brand response:", text);

  if (!res.ok) {
    throw new Error(`ブランド情報の取得に失敗しました (status ${res.status})`);
  }

  const data = JSON.parse(text) as BrandResponse;
  const name = (data.name ?? "").trim();
  if (!name) {
    throw new Error("ブランド名が取得できませんでした。");
  }
  return name;
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
    } catch (e) {
      // 取得失敗時は ID をそのまま表示用に返しておく
      // eslint-disable-next-line no-console
      console.warn(
        "[InvitationRepositoryHTTP] failed to fetch brand name for id:",
        id,
        e,
      );
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
  const url = `${API_BASE}/api/invitation/validate`;
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] validating invitation:", url);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ token }),
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] validate response:", text);

  if (!res.ok) {
    let msg = `招待の検証に失敗しました (status ${res.status})`;
    try {
      const errJson = JSON.parse(text) as { error?: string };
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }

  const data = JSON.parse(text) as ValidateResponse;
  return data;
}

// ------------------------------
// completeInvitationOnBackend (POST /api/invitation/complete)
// ------------------------------
export async function completeInvitationOnBackend(
  payload: CompleteInvitationBackendPayload,
): Promise<void> {
  const url = `${API_BASE}/api/invitation/complete`;
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] complete payload:", payload);

  const res = await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  const text = await res.text();
  // eslint-disable-next-line no-console
  console.log("[InvitationRepositoryHTTP] complete response:", text);

  if (!res.ok) {
    let msg = `招待の完了処理に失敗しました (status ${res.status})`;
    try {
      const errJson = JSON.parse(text) as { error?: string };
      if (errJson.error) msg = errJson.error;
    } catch {
      // ignore
    }
    throw new Error(msg);
  }
}
