//frontend\console\list\src\infrastructure\http\list\signedUrl.ts
import type { SignedListImageUploadDTO } from "./types";
import { buildPublicGcsUrl } from "./gcsUrl";
import { s } from "./string";

/**
 * ✅ Signed URL レスポンスを “呼び出し側が使える形” に正規化する
 * - backend: uploadUrl/publicUrl/objectPath/id...
 * - legacy: signedUrl/publicUrl/objectPath...
 */
export function normalizeSignedListImageUploadDTO(raw: any): SignedListImageUploadDTO {
  const id = s(raw?.id) || s(raw?.ID) || undefined;
  const bucket = s(raw?.bucket) || s(raw?.Bucket) || undefined;

  const objectPath =
    s(raw?.objectPath) ||
    s(raw?.ObjectPath) ||
    s(raw?.path) ||
    s(raw?.Path) ||
    s(raw?.id) || // backend では id=objectPath のことがある
    "";

  const signedUrl =
    s(raw?.signedUrl) ||
    s(raw?.signedURL) ||
    s(raw?.uploadUrl) ||
    s(raw?.uploadURL) ||
    "";

  const publicUrl =
    s(raw?.publicUrl) || s(raw?.publicURL) || s(raw?.url) || s(raw?.URL) || "";

  // もし publicUrl が無いなら bucket+objectPath から組み立て（表示用）
  const builtPublicUrl = publicUrl || buildPublicGcsUrl(bucket || "", objectPath);

  const expiresAt = s(raw?.expiresAt) || s(raw?.ExpiresAt) || undefined;
  const contentType = s(raw?.contentType) || s(raw?.ContentType) || undefined;

  const size = Number(raw?.size);
  const displayOrder = Number(raw?.displayOrder);
  const fileName = s(raw?.fileName) || s(raw?.FileName) || undefined;

  if (!objectPath || !signedUrl) {
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
