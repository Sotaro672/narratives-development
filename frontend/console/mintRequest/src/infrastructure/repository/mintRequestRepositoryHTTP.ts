// frontend/console/mintRequest/src/infrastructure/repository/mintRequestRepositoryHTTP.ts

// Firebase Auth から ID トークンを取得
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";
import type { InspectionBatchDTO } from "../api/mintRequestApi";

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
// HTTP Repository (inspections)
// ===============================

/**
 * inspections の一覧を取得して、そのまま InspectionBatchDTO[] を返す。
 * バックエンド側では /products/inspections が inspections コレクション
 * を参照している想定。
 */
export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const res = await fetch(`${API_BASE}/products/inspections`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    throw new Error(
      `Failed to fetch inspections: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  if (!json) return [];
  return json;
}

/**
 * 個別の productionId に紐づく InspectionBatch を取得。
 *
 * 専用の /products/inspections/{id} エンドポイントは作らず、
 * 一覧を取得してから front 側で productionId で絞り込む。
 */
export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  const batches = await fetchInspectionBatchesHTTP();
  const found = batches.find((b) => b.productionId === trimmed) ?? null;
  return found;
}
