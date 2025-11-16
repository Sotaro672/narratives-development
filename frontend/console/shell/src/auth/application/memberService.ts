// frontend/console/shell/src/auth/application/memberService.ts
/// <reference types="vite/client" />

import type { MemberDTO } from "../domain/entity/member";
import { auth } from "../infrastructure/config/firebaseClient";

// -------------------------------
// Backend base URL（useMemberDetail と同じ構成）
// -------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

// 最終的に使うベース URL
const API_BASE = ENV_BASE || FALLBACK_BASE;

// -------------------------------
// Fetch currentMember（useMemberDetail と同じ API_BASE & 防御）
// -------------------------------
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

  // HTML が返ってきていないかチェック（env ミス検出用）
  const ct = res.headers.get("Content-Type") ?? "";
  if (!ct.includes("application/json")) {
    throw new Error(
      `currentMember API が JSON を返していません (content-type=${ct}). ` +
        `VITE_BACKEND_BASE_URL または API_BASE=${API_BASE} を確認してください。`,
    );
  }

  const raw = (await res.json()) as any;
  if (!raw) return null;

  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";
  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  const firstName = noFirst ? null : (raw.firstName as string);
  const lastName = noLast ? null : (raw.lastName as string);

  const full = `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    id: raw.id ?? uid,
    firstName,
    lastName,
    email: raw.email ?? null,
    companyId: raw.companyId ?? "",
    fullName: full,
  };
}
