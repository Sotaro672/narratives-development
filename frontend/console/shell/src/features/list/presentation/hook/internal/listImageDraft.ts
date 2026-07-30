// frontend/console/shell/src/features/list/presentation/hook/internal/listImageDraft.ts

export type DraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

export function cloneDraftImagesFromUrls(
  urls: string[],
): DraftImage[] {
  if (!Array.isArray(urls)) {
    return [];
  }

  return urls
    .map((url) =>
      String(url ?? "").trim(),
    )
    .filter(Boolean)
    .map((url) => ({
      url,
      isNew: false,
    }));
}

export function revokeDraftBlobUrls(
  items: DraftImage[],
): void {
  if (!Array.isArray(items)) {
    return;
  }

  for (const item of items) {
    if (
      !item.isNew ||
      typeof item.url !== "string" ||
      !item.url.startsWith("blob:")
    ) {
      continue;
    }

    try {
      URL.revokeObjectURL(item.url);
    } catch {
      // Blob URLの解放失敗は無視する。
    }
  }
}

export function fileKey(
  file: File,
): string {
  return [
    file.name,
    file.size,
    file.lastModified,
  ].join("__");
}

export function isImageFile(
  file: File,
): boolean {
  return file.type.startsWith("image/");
}