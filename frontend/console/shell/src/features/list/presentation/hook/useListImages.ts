// frontend/console/shell/src/features/list/presentation/hook/useListImages.ts

import * as React from "react";

import { validateImageForStorage } from "../../../../shared/storage/imageStoragePolicy";

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

export type UseListImagesArgs = {
  isEdit: boolean;
  saving: boolean;
  initialImages: readonly ListImageSource[];
};

export type UseListImagesResult = {
  draftImages: DraftImage[];
  imageUrls: string[];
  onAddImages: (files: File[]) => void;
  onRemoveImageAt: (index: number) => void;
  onClearImages: () => void;
  releaseDraftBlobUrls: () => void;
};

function cloneDraftImagesFromImages(
  images: readonly ListImageSource[],
): DraftImage[] {
  return images.map((image) => ({
    id: image.id,
    url: image.url,
    isNew: false,
  }));
}

function revokeDraftBlobUrls(
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

function fileKey(file: File): string {
  return [
    file.name,
    file.size,
    file.lastModified,
  ].join("__");
}

export function useListImages(
  args: UseListImagesArgs,
): UseListImagesResult {
  const {
    isEdit,
    saving,
    initialImages,
  } = args;

  const [draftImages, setDraftImages] =
    React.useState<DraftImage[]>(
      () => cloneDraftImagesFromImages(initialImages),
    );

  React.useEffect(() => {
    if (isEdit) {
      return;
    }

    setDraftImages(
      cloneDraftImagesFromImages(initialImages),
    );
  }, [
    isEdit,
    initialImages,
  ]);

  const onAddImages = React.useCallback(
    (files: File[]) => {
      if (
        !isEdit ||
        saving ||
        files.length === 0
      ) {
        return;
      }

      for (const file of files) {
        const validation = validateImageForStorage(
          file,
          "listImage",
        );

        if (!validation.valid) {
          window.alert(validation.reason);
          return;
        }
      }

      setDraftImages((previousImages) => {
        const existingFileKeys = new Set(
          previousImages
            .filter(
              (
                image,
              ): image is DraftImage & { file: File } =>
                image.isNew &&
                image.file !== undefined,
            )
            .map((image) => fileKey(image.file)),
        );

        const newImages: DraftImage[] = [];

        for (const file of files) {
          const key = fileKey(file);

          if (existingFileKeys.has(key)) {
            continue;
          }

          existingFileKeys.add(key);

          newImages.push({
            url: URL.createObjectURL(file),
            file,
            isNew: true,
          });
        }

        return [
          ...previousImages,
          ...newImages,
        ];
      });
    },
    [
      isEdit,
      saving,
    ],
  );

  const onRemoveImageAt = React.useCallback(
    (index: number) => {
      if (!isEdit || saving) {
        return;
      }

      setDraftImages((previousImages) => {
        if (
          index < 0 ||
          index >= previousImages.length
        ) {
          return previousImages;
        }

        const target = previousImages[index];

        if (
          target.isNew &&
          target.url.startsWith("blob:")
        ) {
          try {
            URL.revokeObjectURL(target.url);
          } catch {
            // Blob URLの解放失敗は無視する。
          }
        }

        return [
          ...previousImages.slice(0, index),
          ...previousImages.slice(index + 1),
        ];
      });
    },
    [
      isEdit,
      saving,
    ],
  );

  const onClearImages = React.useCallback(() => {
    if (!isEdit || saving) {
      return;
    }

    setDraftImages((previousImages) => {
      revokeDraftBlobUrls(previousImages);
      return [];
    });
  }, [
    isEdit,
    saving,
  ]);

  const releaseDraftBlobUrls = React.useCallback(() => {
    revokeDraftBlobUrls(draftImages);
  }, [draftImages]);

  const imageUrls = React.useMemo(
    () => draftImages.map((image) => image.url),
    [draftImages],
  );

  return {
    draftImages,
    imageUrls,
    onAddImages,
    onRemoveImageAt,
    onClearImages,
    releaseDraftBlobUrls,
  };
}