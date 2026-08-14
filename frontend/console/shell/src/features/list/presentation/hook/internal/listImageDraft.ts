// frontend/console/shell/src/features/list/presentation/hook/internal/listImageDraft.ts

export type ListImageSource = {
  id: string;
  url: string;
  displayOrder: number;
};

export type DraftImage = {
  id?: string;
  url: string;
  isNew: boolean;
  file?: File;
};

export function cloneDraftImagesFromImages(
  images: readonly ListImageSource[],
): DraftImage[] {
  return images.map((image) => ({
    id: image.id,
    url: image.url,
    isNew: false,
  }));
}

export function revokeDraftBlobUrls(
  items: readonly DraftImage[],
): void {
  for (const item of items) {
    if (!item.isNew || !item.url.startsWith("blob:")) {
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