// frontend/console/brand/src/application/assignedMemberService.ts

/// <reference types="vite/client" />

import { getAuthHeaders } from "../../../shell/src/auth/application/authService";

export type AssignedMember = {
  id: string;
  name: string;
  email?: string;
  status?: string;
};

/** 「姓 名」表示用のヘルパー */
function formatLastFirst(
  last?: string | null,
  first?: string | null,
): string {
  const ln = (last ?? "").trim();
  const fn = (first ?? "").trim();
  if (ln && fn) return `${ln} ${fn}`;
  if (ln) return ln;
  if (fn) return fn;
  return "";
}

// backend base URL
// 1. .env の VITE_BACKEND_BASE_URL を使う
// 2. 設定がなければ Cloud Run の URL にフォールバック
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const API_BASE =
  ENV_BASE || "https://narratives-backend-871263659099.asia-northeast1.run.app";

/**
 * brandId に割り当てられているメンバー一覧を取得する。
 * backend の /members エンドポイントに brandIds フィルタ付きで問い合わせる。
 */
export async function fetchAssignedMembers(
  brandId: string,
): Promise<AssignedMember[]> {
  const trimmed = brandId.trim();
  if (!trimmed) {
    return [];
  }

  const headers = await getAuthHeaders();

  // ひとまず 1ページ目・最大100件を想定
  const params = new URLSearchParams({
    brandIds: trimmed,
    page: "1",
    perPage: "100",
    sort: "updatedAt",
    order: "desc",
  });

  const url = `${API_BASE}/members?${params.toString()}`;
  // eslint-disable-next-line no-console
  console.log("[assignedMemberService] GET", url);

  const res = await fetch(url, { headers });

  const ct = res.headers.get("content-type") ?? "";
  if (!ct.includes("application/json")) {
    const text = await res.text().catch(() => "");
    throw new Error(`Unexpected content-type: ${ct}\n${text.slice(0, 200)}`);
  }

  if (!res.ok) {
    const text = await res.text().catch(() => `HTTP ${res.status}`);
    throw new Error(text);
  }

  const data: any = await res.json();

  // backend が PageResult か生配列か、両方に対応
  const items: any[] = Array.isArray(data) ? data : data.items ?? [];

  const mapped: AssignedMember[] = items.map((m) => {
    const id = String(m.id ?? "").trim();
    return {
      id,
      name: formatLastFirst(m.lastName, m.firstName) || id,
      email: m.email ?? undefined,
      status: m.status ? String(m.status) : undefined,
    };
  });

  // デバッグログ
  // eslint-disable-next-line no-console
  console.log("[assignedMemberService] fetchAssignedMembers result:", mapped);

  return mapped;
}
