// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts 

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { MintRequestDTO } from "../api/mintRequestApi";

// 🔙 BACKEND の BASE URL
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

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

// ===============================
// HTTP Repository (mintRequests)
// ===============================

/**
 * 現在ログイン中の companyId に紐づく MintRequest 一覧を取得する。
 *
 * バックエンド側:
 *   - AuthMiddleware が context に companyId を注入
 *   - MintRequestUsecase.ListByCurrentCompany(ctx) が
 *       1) productBlueprint (companyId 絞り込み)
 *       2) production (productBlueprintId 絞り込み)
 *       3) mintRequests  (ListByProductionIDs)
 *     を内部で呼び出す。
 *
 * フロント側は単に GET /mint-requests を叩くだけでよい。
 */
export async function fetchMintRequestsHTTP(): Promise<MintRequestDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const res = await fetch(`${API_BASE}/mint-requests`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch mintRequests: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as MintRequestDTO[] | null | undefined;
  if (!json) return [];
  return json;
}

/**
 * 個別の MintRequest を ID で取得する。
 *   GET /mint-requests/{id}
 */
export async function fetchMintRequestByIdHTTP(
  id: string,
): Promise<MintRequestDTO | null> {
  const idToken = await getIdTokenOrThrow();

  const trimmed = id.trim();
  if (!trimmed) {
    throw new Error("mintRequestId が空です");
  }

  const res = await fetch(
    `${API_BASE}/mint-requests/${encodeURIComponent(trimmed)}`,
    {
      method: "GET",
      headers: {
        Authorization: `Bearer ${idToken}`,
        "Content-Type": "application/json",
      },
    },
  );

  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error(
      `Failed to fetch mintRequest: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as MintRequestDTO;
  return json;
}
