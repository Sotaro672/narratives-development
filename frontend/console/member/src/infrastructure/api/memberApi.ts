// frontend/member/src/infrastructure/api/memberApi.ts
export type CreateMemberRequest = {
  id: string;
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  permissions: string[];
  assignedBrands?: string[] | null;
};

export type MemberResponse = {
  id: string;
  first_name?: string;
  last_name?: string;
  first_name_kana?: string | null;
  last_name_kana?: string | null;
  email?: string;
  permissions: string[];
  assignedBrands?: string[];
  createdAt: string;
  updatedAt?: string | null;
};

const API_BASE =
  (import.meta as any).env?.VITE_BACKEND_BASE_URL?.replace(/\/+$/, "") ?? "";

export async function createMember(req: CreateMemberRequest): Promise<MemberResponse> {
  const res = await fetch(`${API_BASE}/members`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      id: req.id,
      firstName: req.firstName ?? "",
      lastName: req.lastName ?? "",
      firstNameKana: req.firstNameKana ?? "",
      lastNameKana: req.lastNameKana ?? "",
      email: req.email ?? "",
      permissions: req.permissions,
      assignedBrands: req.assignedBrands ?? [],
    }),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(
      `failed to create member: ${res.status} ${res.statusText} ${text}`
    );
  }

  return (await res.json()) as MemberResponse;
}
