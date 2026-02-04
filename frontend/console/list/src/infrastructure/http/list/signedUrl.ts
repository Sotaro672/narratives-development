// frontend/console/list/src/infrastructure/http/list/signedUrl.ts
import type { SignedListImageUploadDTO } from "./types";
import { buildPublicGcsUrl } from "./gcsUrl";
import { s } from "./string";

/**
 * ✅ Signed URL レスポンスを “呼び出し側が使える形” に正規化する
 * - backend: uploadUrl/publicUrl/objectPath/id...
 * - legacy: signedUrl/publicUrl/objectPath...
 *
 * NOTE:
 * - types.ts 側で id/bucket/objectPath/signedUrl が必須(string)なので
 *   ここでも必ず string を返す（undefined は返さない）
 */
export function normalizeSignedListImageUploadDTO(raw: any): SignedListImageUploadDTO {
  // 必須: string に寄せる（undefined にしない）
  const id = s(raw?.id || raw?.ID);
  const bucket = s(raw?.bucket || raw?.Bucket);

  const objectPath =
    s(raw?.objectPath) ||
    s(raw?.ObjectPath) ||
    s(raw?.path) ||
    s(raw?.Path) ||
    ""; // 旧式互換で id=objectPath のケースはここでは採らない（事故防止）

  // ✅ 呼び出し側は signedUrl を期待するので uploadUrl を吸収
  const signedUrl =
    s(raw?.signedUrl) ||
    s(raw?.signedURL) ||
    s(raw?.uploadUrl) ||
    s(raw?.uploadURL) ||
    "";

  const publicUrl = s(raw?.publicUrl) || s(raw?.publicURL) || "";

  // もし publicUrl が無いなら bucket+objectPath から組み立て（表示用）
  const builtPublicUrl = publicUrl || (bucket && objectPath ? buildPublicGcsUrl(bucket, objectPath) : "");

  const expiresAt = s(raw?.expiresAt) || s(raw?.ExpiresAt) || undefined;
  const contentType = s(raw?.contentType) || s(raw?.ContentType) || undefined;

  const size = Number(raw?.size);
  const displayOrder = Number(raw?.displayOrder);
  const fileName = s(raw?.fileName) || s(raw?.FileName) || undefined;

  // ✅ 必須チェック（types.ts の必須項目に合わせる）
  if (!id || !bucket || !objectPath || !signedUrl) {
    // inventory 側のエラーハンドリングが msg === "signed_url_response_invalid" を見てるので合わせる
    throw new Error("signed_url_response_invalid");
  }

  return {
    id,
    bucket,
    objectPath,
    signedUrl,
    publicUrl: builtPublicUrl || undefined,
    expiresAt,
    contentType,
    size: Number.isFinite(size) ? size : undefined,
    displayOrder: Number.isFinite(displayOrder) ? displayOrder : undefined,
    fileName,
  };
}
