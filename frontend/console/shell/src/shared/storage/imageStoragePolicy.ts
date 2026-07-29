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

export function isAllowedImageMimeType(
  contentType: string,
): boolean {
  return imageStoragePolicy.allowedMimeTypes.includes(
    contentType.toLowerCase(),
  );
}

export function validateImageForStorage(
  file: File,
  target: ImageStorageTarget,
): ImageValidationResult {
  if (!file) {
    return {
      valid: false,
      reason: "画像ファイルが選択されていません。",
    };
  }

  const contentType = file.type.toLowerCase();

  if (!isAllowedImageMimeType(contentType)) {
    return {
      valid: false,
      reason:
        "画像形式はJPEG、PNG、WebPのみ使用できます。",
    };
  }

  const maxBytes = getImageMaxBytes(target);

  if (
    !Number.isFinite(file.size) ||
    file.size <= 0
  ) {
    return {
      valid: false,
      reason: "画像ファイルのサイズが不正です。",
    };
  }

  if (file.size > maxBytes) {
    const maxMegabytes = maxBytes / 1024 / 1024;

    return {
      valid: false,
      reason: `画像サイズは${maxMegabytes}MB以下にしてください。`,
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
  const result = validateImageForStorage(
    file,
    target,
  );

  if (!result.valid) {
    throw new Error(result.reason);
  }
}