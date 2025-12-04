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
// 共通: Firebase トークン取得 + companyId デバッグ
// ---------------------------------------------------------
async function getIdTokenOrThrow(): Promise<string> {
  const user = auth.currentUser;
  if (!user) {
    throw new Error("ログイン情報が見つかりません（未ログイン）");
  }

  const idToken = await user.getIdToken();

  // ★★★ デバッグ: ID トークンの payload を decode して companyId を確認 ★★★
  try {
    const decoded = JSON.parse(atob(idToken.split(".")[1]));
    console.log("[MintRequestRepo] ID Token claims:", decoded);
    console.log("[MintRequestRepo] companyId from claims:", decoded.companyId);
  } catch (e) {
    console.warn("[MintRequestRepo] Failed to decode ID token:", e);
  }

  return idToken;
}

// ===============================
// HTTP Repository (inspections)
// ===============================

/**
 * 現在ログイン中の companyId を起点に、
 * /mint/inspections から inspections の一覧を取得する。
 */
export async function fetchInspectionBatchesHTTP(): Promise<InspectionBatchDTO[]> {
  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/mint/inspections`;

  // ★★★ バックエンドへ渡すURLの確認 ★★★
  console.log("[MintRequestRepo] Fetch URL (mint inspections):", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (!res.ok) {
    console.error("[MintRequestRepo] Fetch failed:", res.status, res.statusText);
    throw new Error(
      `Failed to fetch inspections (mint): ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO[] | null | undefined;
  if (!json) return [];

  return json;
}

/**
 * 個別 productionId の InspectionBatch を取得
 * （こちらは従来どおり /products/inspections?productionId=... を使用）
 */
export async function fetchInspectionByProductionIdHTTP(
  productionId: string,
): Promise<InspectionBatchDTO | null> {
  const trimmed = productionId.trim();
  if (!trimmed) {
    throw new Error("productionId が空です");
  }

  const idToken = await getIdTokenOrThrow();

  const url = `${API_BASE}/products/inspections?productionId=${encodeURIComponent(
    trimmed,
  )}`;

  // ★★★ バックエンドへ渡すURL（productionId付き）をログ出力 ★★★
  console.log("[MintRequestRepo] Fetch URL (by productionId):", url);

  const res = await fetch(url, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${idToken}`,
      "Content-Type": "application/json",
    },
  });

  if (res.status === 404) {
    console.log(
      "[MintRequestRepo] No inspection batch found for productionId:",
      trimmed,
    );
    return null;
  }

  if (!res.ok) {
    console.error("[MintRequestRepo] Fetch failed:", res.status, res.statusText);
    throw new Error(
      `Failed to fetch inspection by productionId: ${res.status} ${res.statusText}`,
    );
  }

  const json = (await res.json()) as InspectionBatchDTO | null | undefined;
  if (!json) return null;

  return json;
}
