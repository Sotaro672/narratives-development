// このファイルは自動生成です。直接編集しないでください。

export const BRAND_IMAGE_ALLOWED_MIME_TYPES = [
  "image/jpeg",
  "image/png",
  "image/webp"
] as const;

export type BrandImageMimeType =
  (typeof BRAND_IMAGE_ALLOWED_MIME_TYPES)[number];

export const BRAND_IMAGE_MAX_BYTES = {
  brandIcon: 2097152,
  brandBackgroundImage: 5242880,
} as const;

export type BrandImageTarget =
  keyof typeof BRAND_IMAGE_MAX_BYTES;
