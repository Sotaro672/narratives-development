//frontend\console\shell\src\features\brand\application\brandImageValidation.ts
import {
  BRAND_IMAGE_ALLOWED_MIME_TYPES,
  BRAND_IMAGE_MAX_BYTES,
  type BrandImageTarget,
} from "../config/brandImagePolicy.generated";

export type BrandImageValidationResult =
  | {
      valid: true;
    }
  | {
      valid: false;
      message: string;
    };

export function validateBrandImage(
  file: File,
  target: BrandImageTarget,
): BrandImageValidationResult {
  const allowedMimeTypes: readonly string[] =
    BRAND_IMAGE_ALLOWED_MIME_TYPES;

  if (!allowedMimeTypes.includes(file.type)) {
    return {
      valid: false,
      message:
        "JPEG、PNG、WebP形式の画像を選択してください。",
    };
  }

  const maxBytes = BRAND_IMAGE_MAX_BYTES[target];

  if (file.size > maxBytes) {
    const maxMegabytes = maxBytes / (1024 * 1024);

    return {
      valid: false,
      message: `画像サイズは${maxMegabytes}MB以下にしてください。`,
    };
  }

  return {
    valid: true,
  };
}