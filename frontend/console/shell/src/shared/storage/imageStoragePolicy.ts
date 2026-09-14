// frontend/console/shell/src/shared/storage/imageStoragePolicy.ts

import imagePolicyJson from "../config/storageImagePolicy.json";

export type ImageStorageTarget = keyof typeof imagePolicyJson.targets;

type ImageTargetPolicy = {
  maxBytes: number;
};

type ImageStoragePolicy = {
  allowedMimeTypes: string[];
  defaultMaxBytes: number;
  targets: Record<string, ImageTargetPolicy>;
};

const imageStoragePolicy =
  imagePolicyJson as ImageStoragePolicy;

const DEFAULT_IMAGE_CONTENT_TYPE =
  "application/octet-stream";

export const IMAGE_STORAGE_ACCEPT =
  imageStoragePolicy.allowedMimeTypes.join(",");

export type ImageValidationResult =
  | {
      valid: true;
    }
  | {
      valid: false;
      reason: string;
    };

export function getImageMaxBytes(
  target: ImageStorageTarget,
): number {
  return (
    imageStoragePolicy.targets[target]?.maxBytes ??
    imageStoragePolicy.defaultMaxBytes
  );
}

export function getImageContentType(
  file: File,
): string {
  return (
    String(file?.type ?? "")
      .trim()
      .toLowerCase() ||
    DEFAULT_IMAGE_CONTENT_TYPE
  );
}

export function isAllowedImageMimeType(
  contentType: string,
): boolean {
  const normalizedContentType =
    String(contentType ?? "")
      .trim()
      .toLowerCase();

  return imageStoragePolicy.allowedMimeTypes.includes(
    normalizedContentType,
  );
}

export function validateImageForStorage(
  file: File,
  target: ImageStorageTarget,
): ImageValidationResult {
  if (!file) {
    return {
      valid: false,
      reason:
        "画像ファイルが選択されていません。",
    };
  }

  const fileName =
    String(file.name ?? "").trim();

  if (!fileName) {
    return {
      valid: false,
      reason:
        "画像ファイル名を取得できません。別の画像を選択してください。",
    };
  }

  if (
    !Number.isFinite(file.size) ||
    file.size <= 0
  ) {
    return {
      valid: false,
      reason:
        `「${fileName}」のデータが空または不正です。別の画像を選択してください。`,
    };
  }

  const contentType =
    getImageContentType(file);

  if (
    !isAllowedImageMimeType(
      contentType,
    )
  ) {
    return {
      valid: false,
      reason:
        `「${fileName}」は対応していない画像形式です。JPEG、PNG、WebP、AVIF形式の画像を選択してください。`,
    };
  }

  const maxBytes =
    getImageMaxBytes(target);

  if (file.size > maxBytes) {
    const maxMegabytes =
      maxBytes / 1024 / 1024;

    return {
      valid: false,
      reason:
        `「${fileName}」のファイルサイズが${maxMegabytes}MBを超えています。${maxMegabytes}MB以下の画像を選択してください。`,
    };
  }

  return {
    valid: true,
  };
}

export function assertImageForStorage(
  file: File,
  target: ImageStorageTarget,
): void {
  const result =
    validateImageForStorage(
      file,
      target,
    );

  if (!result.valid) {
    throw new Error(
      result.reason,
    );
  }
}