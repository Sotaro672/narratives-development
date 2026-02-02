//frontend\console\list\src\infrastructure\http\list\config.ts
/**
 * Backend base URL
 */
const ENV_BASE =
  ((import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined)?.replace(
    /\/+$/g,
    "",
  ) ?? "";

const FALLBACK_BASE =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

export const API_BASE = ENV_BASE || FALLBACK_BASE;

/**
 * ✅ ListImage bucket (public access想定: https://storage.googleapis.com/{bucket}/{objectPath})
 * - backend 側の fallback と合わせる
 * - 将来 env を増やす場合に備えて VITE_LIST_IMAGE_BUCKET も見ておく
 */
const ENV_LIST_IMAGE_BUCKET = String(
  (import.meta as any).env?.VITE_LIST_IMAGE_BUCKET ?? "",
).trim();

const FALLBACK_LIST_IMAGE_BUCKET = "narratives-development-list";

export const LIST_IMAGE_BUCKET = ENV_LIST_IMAGE_BUCKET || FALLBACK_LIST_IMAGE_BUCKET;
