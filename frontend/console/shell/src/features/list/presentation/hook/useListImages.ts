// frontend/console/shell/src/features/list/presentation/hook/useListImages.ts

import * as React from "react";

import {
  cloneDraftImagesFromUrls,
  fileKey,
  isImageFile,
  revokeDraftBlobUrls,
  type DraftImage,
} from "./internal/listImageDraft";

export type {
  DraftImage,
} from "./internal/listImageDraft";

export type UseListImagesArgs = {
  isEdit: boolean;
  saving: boolean;
  initialUrls: string[];
};

export type UseListImagesResult = {
  draftImages: DraftImage[];
  imageUrls: string[];

  onAddImages: (
    files: FileList | null,
  ) => void;

  onRemoveImageAt: (
    index: number,
  ) => void;

  onClearImages: () => void;
  releaseDraftBlobUrls: () => void;
};

export function useListImages(
  args: UseListImagesArgs,
): UseListImagesResult {
  const {
    isEdit,
    saving,
    initialUrls,
  } = args;

  const [
    draftImages,
    setDraftImages,
  ] = React.useState<DraftImage[]>(
    cloneDraftImagesFromUrls(
      initialUrls,
    ),
  );

  React.useEffect(() => {
    if (isEdit) {
      return;
    }

    setDraftImages(
      cloneDraftImagesFromUrls(
        initialUrls,
      ),
    );
  }, [
    isEdit,
    initialUrls,
  ]);

  const addFiles =
    React.useCallback(
      (files: File[]) => {
        if (
          !isEdit ||
          saving
        ) {
          return;
        }

        const incomingFiles =
          (
            Array.isArray(files)
              ? files
              : []
          )
            .filter(Boolean)
            .filter(isImageFile);

        if (
          incomingFiles.length === 0
        ) {
          return;
        }

        setDraftImages(
          (previousImages) => {
            const currentImages =
              Array.isArray(
                previousImages,
              )
                ? previousImages
                : [];

            const existingFileKeys =
              new Set(
                currentImages
                  .filter(
                    (
                      image,
                    ): image is DraftImage & {
                      file: File;
                    } =>
                      image.isNew &&
                      Boolean(
                        image.file,
                      ),
                  )
                  .map((image) =>
                    fileKey(
                      image.file,
                    ),
                  ),
              );

            const newImages:
              DraftImage[] = [];

            for (
              const file
              of incomingFiles
            ) {
              const key =
                fileKey(file);

              if (
                existingFileKeys.has(
                  key,
                )
              ) {
                continue;
              }

              existingFileKeys.add(
                key,
              );

              newImages.push({
                url:
                  URL.createObjectURL(
                    file,
                  ),
                file,
                isNew: true,
              });
            }

            return [
              ...currentImages,
              ...newImages,
            ];
          },
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const onAddImages =
    React.useCallback(
      (
        files:
          FileList |
          null,
      ) => {
        if (
          !files ||
          files.length === 0
        ) {
          return;
        }

        addFiles(
          Array.from(
            files,
          ).filter(Boolean),
        );
      },
      [addFiles],
    );

  const onRemoveImageAt =
    React.useCallback(
      (
        index: number,
      ) => {
        if (
          !isEdit ||
          saving
        ) {
          return;
        }

        setDraftImages(
          (previousImages) => {
            const currentImages =
              Array.isArray(
                previousImages,
              )
                ? previousImages
                : [];

            if (
              index < 0 ||
              index >=
                currentImages.length
            ) {
              return currentImages;
            }

            const target =
              currentImages[index];

            if (
              target?.isNew &&
              target.url.startsWith(
                "blob:",
              )
            ) {
              try {
                URL.revokeObjectURL(
                  target.url,
                );
              } catch {
                // Blob URLの解放失敗は無視する。
              }
            }

            return [
              ...currentImages.slice(
                0,
                index,
              ),
              ...currentImages.slice(
                index + 1,
              ),
            ];
          },
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const onClearImages =
    React.useCallback(
      () => {
        if (
          !isEdit ||
          saving
        ) {
          return;
        }

        setDraftImages(
          (
            previousImages,
          ) => {
            const currentImages =
              Array.isArray(
                previousImages,
              )
                ? previousImages
                : [];

            revokeDraftBlobUrls(
              currentImages,
            );

            return [];
          },
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const releaseDraftBlobUrls =
    React.useCallback(
      () => {
        revokeDraftBlobUrls(
          draftImages,
        );
      },
      [draftImages],
    );

  const imageUrls =
    React.useMemo(
      () =>
        draftImages
          .map((image) =>
            String(
              image.url ?? "",
            ).trim(),
          )
          .filter(Boolean),
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