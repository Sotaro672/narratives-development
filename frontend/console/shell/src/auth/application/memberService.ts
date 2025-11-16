// frontend/console/shell/src/auth/application/memberService.ts
/// <reference types="vite/client" />

import type { MemberDTO } from "../domain/entity/member";
import { auth } from "../infrastructure/config/firebaseClient";

const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

const API_BASE = ENV_BASE || FALLBACK_BASE;

// 既存: fetchCurrentMember はそのまま

export async function fetchCurrentMember(uid: string): Promise<MemberDTO | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const url = `${API_BASE}/members/${encodeURIComponent(uid)}`;
  console.log("[memberService] fetchCurrentMember uid:", uid, "GET", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn(
      "[memberService] fetchCurrentMember failed:",
      res.status,
      res.statusText,
      text,
    );
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    throw new Error(
      `currentMember API が JSON を返していません (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または API_BASE=${API_BASE} を確認してください。`,
    );
  }

  const raw = (await res.json()) as any;
  if (!raw) return null;

  const firstName =
    raw.firstName && String(raw.firstName).trim() !== ""
      ? String(raw.firstName)
      : null;
  const lastName =
    raw.lastName && String(raw.lastName).trim() !== ""
      ? String(raw.lastName)
      : null;

  const firstNameKana =
    raw.firstNameKana && String(raw.firstNameKana).trim() !== ""
      ? String(raw.firstNameKana)
      : null;

  const lastNameKana =
    raw.lastNameKana && String(raw.lastNameKana).trim() !== ""
      ? String(raw.lastNameKana)
      : null;

  const full = `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    id: raw.id ?? uid,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email: raw.email ?? null,
    companyId: raw.companyId ?? "",
    fullName: full,
  };
}

// ★ 追加: プロフィール更新
export type UpdateMemberProfileInput = {
  id: string;
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
};

export async function updateCurrentMemberProfile(
  input: UpdateMemberProfileInput,
): Promise<MemberDTO | null> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) return null;

  const url = `${API_BASE}/members/${encodeURIComponent(input.id)}`;
  console.log("[memberService] updateCurrentMemberProfile PATCH", url, input);

  const res = await fetch(url, {
    method: "PATCH", // Backend 側のメソッドに合わせる（PUT なら PUT に変更）
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      firstName: input.firstName,
      lastName: input.lastName,
      firstNameKana: input.firstNameKana,
      lastNameKana: input.lastNameKana,
    }),
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    console.warn(
      "[memberService] updateCurrentMemberProfile failed:",
      res.status,
      res.statusText,
      text,
    );
    return null;
  }

  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    // 成功だけどボディ無し、などなら null でもよい
    return null;
  }

  const raw = (await res.json()) as any;

  const firstName =
    raw.firstName && String(raw.firstName).trim() !== ""
      ? String(raw.firstName)
      : null;
  const lastName =
    raw.lastName && String(raw.lastName).trim() !== ""
      ? String(raw.lastName)
      : null;

  const firstNameKana =
    raw.firstNameKana && String(raw.firstNameKana).trim() !== ""
      ? String(raw.firstNameKana)
      : null;

  const lastNameKana =
    raw.lastNameKana && String(raw.lastNameKana).trim() !== ""
      ? String(raw.lastNameKana)
      : null;

  const full = `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    id: raw.id ?? input.id,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email: raw.email ?? null,
    companyId: raw.companyId ?? "",
    fullName: full,
  };
}
