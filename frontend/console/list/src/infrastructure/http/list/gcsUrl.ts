//frontend\console\list\src\infrastructure\http\list\gcsUrl.ts
import { LIST_IMAGE_BUCKET } from "./config";
import { s } from "./string";

/**
 * ✅ objectPath を URL パスとして安全にする
 * - "/" はパス区切りとして残したいのでセグメント単位で encodeURIComponent
 * - 例: "lists/xxx/スクショ (1).png" を安全なURLへ
 */
export function encodeGcsObjectPath(objectPath: string): string {
  const raw = String(objectPath ?? "").trim().replace(/^\/+/, "");
  if (!raw) return "";
  return raw
    .split("/")
    .map((seg) => encodeURIComponent(seg))
    .join("/");
}

export function buildPublicGcsUrl(bucket: string, objectPath: string): string {
  const b = s(bucket) || LIST_IMAGE_BUCKET;
  const opRaw = String(objectPath ?? "").trim().replace(/^\/+/, "");
  const op = encodeGcsObjectPath(opRaw);
  if (!b || !op) return "";
  return `https://storage.googleapis.com/${b}/${op}`;
}
