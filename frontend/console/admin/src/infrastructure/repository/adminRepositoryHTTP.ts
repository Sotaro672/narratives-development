// frontend/console/admin/src/infrastructure/repository/adminRepositoryHTTP.ts

// Admin モジュール用 HTTP リポジトリ
// - /members/by-company から companyId 配下のメンバー一覧を取得
// - backend 側で ListMembersByCompanyID + displayName を付与して返している想定

import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { Member } from "../../../../member/src/domain/entity/member";

// ---------------------------------------------------------
// API BASE URL 設定（他モジュールと同様パターン）
// ---------------------------------------------------------
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

// ---------------------------------------------------------
// 型定義
// ---------------------------------------------------------

// backend の /members/by-company が返す 1 件分の想定型
// Member に displayName が追加されている
export type MemberWithDisplayName = Member & {
  displayName?: string;
};

// ---------------------------------------------------------
// 共通: Firebase トークン取得
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }
  return user.getIdToken();
}

// ---------------------------------------------------------
// /members/by-company 取得
// ---------------------------------------------------------

/**
 * 現在ログイン中ユーザーの companyId コンテキストで
 * /members/by-company を叩き、displayName 付き Member 配列を取得する。
 *
 * backend 側:
 *   - AuthMiddleware で currentMember / companyId を解決
 *   - ListMembersByCompanyID により companyId 配下のメンバー一覧を取得
 *   - 各要素に displayName を付与して JSON 配列で返却
 */
export async function fetchMembersByCompany(): Promise<MemberWithDisplayName[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/members/by-company`;

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    // ステータスコード付きで投げておくとデバッグしやすい
    throw new Error(
      `[AdminRepositoryHTTP] GET /members/by-company failed: ${res.status} ${res.statusText}`,
    );
  }

  const data = await res.json();

  // 想定どおり配列でなければ空配列にフォールバック
  if (!Array.isArray(data)) {
    // 型安全のため軽く警告を出しておく（必要なら後で console.log 削除可）
    console.warn(
      "[AdminRepositoryHTTP] /members/by-company response is not array:",
      data,
    );
    return [];
  }

  return data as MemberWithDisplayName[];
}
