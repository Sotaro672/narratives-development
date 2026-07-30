// frontend/console/shell/src/features/list/presentation/hook/useListDetail.tsx

import * as React from "react";
import { useParams } from "react-router-dom";

import type { PriceRow } from "../../../inventory/application/listCreate/listCreateService";

import { auth } from "../../../../auth/infrastructure/config/firebaseClient";

import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";

import { saveListDetailChanges } from "../../application/listDetail/listDetailSave.usecase";

import { updatePriceRowPrice } from "../../application/listDetail/listDetailMapper";

import {
  isValidListStatus,
  type ListStatus,
} from "../../../../shared/types/list";

import {
  deriveListDetail,
  loadListDetailDTO,
  resolveListDetailParams,
  type ListDetailDTO,
  type ListDetailRouteParams,
} from "../../application/listDetailService";

export type DraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

export type UseListDetailResult = {
  loading: boolean;
  error: string;

  saving: boolean;
  saveError: string;

  isEdit: boolean;
  onEdit: () => void;
  onCancel: () => void;
  onSave: (payload?: any) => Promise<void>;

  listingTitle: string;
  description: string;

  draftListingTitle: string;
  setDraftListingTitle: React.Dispatch<
    React.SetStateAction<string>
  >;

  draftDescription: string;
  setDraftDescription: React.Dispatch<
    React.SetStateAction<string>
  >;

  status: ListStatus | "";
  draftStatus: ListStatus;
  onToggleStatus: (next: ListStatus) => void;

  productBrandName: string;
  productName: string;

  tokenBrandName: string;
  tokenName: string;

  imageUrls: string[];

  onAddImages: (files: FileList | null) => void;
  onRemoveImageAt: (index: number) => void;
  onClearImages: () => void;

  mainImageIndex: number;
  setMainImageIndex: React.Dispatch<
    React.SetStateAction<number>
  >;

  priceRows: PriceRow[];
  draftPriceRows: PriceRow[];

  onChangePrice: (
    index: number,
    price: number | undefined,
    row: PriceRow,
  ) => void;

  assigneeId: string;
  assigneeName: string;
  draftAssigneeId: string;

  onSelectAssignee: (id: string) => void;

  createdByName: string;
  createdAt: string;

  updatedByName: string;
  updatedAt: string;
};

function clonePriceRows(
  rows: PriceRow[],
): PriceRow[] {
  if (!Array.isArray(rows)) {
    return [];
  }

  return rows.map((row) => ({
    ...row,
  }));
}

function cloneDraftImagesFromUrls(
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

function revokeDraftBlobUrls(
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

function fileKey(
  file: File,
): string {
  return [
    file.name,
    file.size,
    file.lastModified,
  ].join("__");
}

function isImageFile(
  file: File,
): boolean {
  return file.type.startsWith("image/");
}

function useListImages(args: {
  isEdit: boolean;
  saving: boolean;
  initialUrls: string[];
}) {
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
              currentImages[
                index
              ];

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
  };
}

export function useListDetail():
  UseListDetailResult {
  const params =
    useParams<
      ListDetailRouteParams
    >();

  const resolved =
    React.useMemo(
      () =>
        resolveListDetailParams(
          params,
        ),
      [params],
    );

  const {
    listId,
  } = resolved;

  const [
    dto,
    setDTO,
  ] =
    React.useState<
      ListDetailDTO |
      null
    >(null);

  const [
    loading,
    setLoading,
  ] =
    React.useState(
      false,
    );

  const [
    error,
    setError,
  ] =
    React.useState(
      "",
    );

  const cancelledRef =
    useCancelledRef();

  const reload =
    React.useCallback(
      async () => {
        const id =
          String(
            listId ?? "",
          ).trim();

        if (!id) {
          setDTO(null);

          setError(
            "listId がありません（ルートパラメータを確認してください）。",
          );

          return;
        }

        setLoading(true);
        setError("");

        try {
          const data =
            await loadListDetailDTO({
              listId: id,
            });

          if (
            cancelledRef.current
          ) {
            return;
          }

          setDTO(data);
        } catch (
          caughtError
        ) {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setError(
            String(
              caughtError
                instanceof Error
                ? caughtError.message
                : caughtError,
            ),
          );

          setDTO(null);
        } finally {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setLoading(false);
        }
      },
      [
        listId,
        cancelledRef,
      ],
    );

  React.useEffect(() => {
    void reload();
  }, [reload]);

  const derived =
    React.useMemo(
      () =>
        deriveListDetail<
          PriceRow
        >(dto),
      [dto],
    );

  const {
    listingTitle,
    description,
    status,

    productBrandName,
    productName,

    tokenBrandName,
    tokenName,

    imageUrls:
      viewImageUrls,

    priceRows:
      viewPriceRows,

    assigneeId,
    assigneeName,

    createdByName,
    createdAt,

    updatedByName,
    updatedAt,
  } = derived;

  const statusForEdit =
    React.useMemo<
      ListStatus
    >(
      () =>
        status ===
        "listing"
          ? "listing"
          : "suspended",
      [status],
    );

  const [
    isEdit,
    setIsEdit,
  ] =
    React.useState(
      false,
    );

  const [
    draftListingTitle,
    setDraftListingTitle,
  ] =
    React.useState(
      listingTitle,
    );

  const [
    draftDescription,
    setDraftDescription,
  ] =
    React.useState(
      description,
    );

  const [
    draftPriceRows,
    setDraftPriceRows,
  ] =
    React.useState<
      PriceRow[]
    >(
      clonePriceRows(
        viewPriceRows,
      ),
    );

  const [
    draftStatus,
    setDraftStatus,
  ] =
    React.useState<
      ListStatus
    >(
      statusForEdit,
    );

  const [
    draftAssigneeId,
    setDraftAssigneeId,
  ] =
    React.useState(
      assigneeId,
    );

  const [
    saving,
    setSaving,
  ] =
    React.useState(
      false,
    );

  const [
    saveError,
    setSaveError,
  ] =
    React.useState(
      "",
    );

  const images =
    useListImages({
      isEdit,
      saving,
      initialUrls:
        viewImageUrls,
    });

  const resetDraftFromView =
    React.useCallback(
      () => {
        setDraftListingTitle(
          listingTitle,
        );

        setDraftDescription(
          description,
        );

        setDraftPriceRows(
          clonePriceRows(
            viewPriceRows,
          ),
        );

        setDraftStatus(
          statusForEdit,
        );

        setDraftAssigneeId(
          assigneeId,
        );
      },
      [
        listingTitle,
        description,
        viewPriceRows,
        statusForEdit,
        assigneeId,
      ],
    );

  React.useEffect(() => {
    if (isEdit) {
      return;
    }

    resetDraftFromView();
  }, [
    isEdit,
    resetDraftFromView,
  ]);

  const onEdit =
    React.useCallback(
      () => {
        resetDraftFromView();
        setSaveError("");
        setIsEdit(true);
      },
      [
        resetDraftFromView,
      ],
    );

  const onCancel =
    React.useCallback(
      () => {
        revokeDraftBlobUrls(
          images.draftImages,
        );

        resetDraftFromView();
        setSaveError("");
        setIsEdit(false);
      },
      [
        images.draftImages,
        resetDraftFromView,
      ],
    );

  const onToggleStatus =
    React.useCallback(
      (
        next:
          ListStatus,
      ) => {
        if (
          !isEdit ||
          saving
        ) {
          return;
        }

        setDraftStatus(
          next,
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const onSelectAssignee =
    React.useCallback(
      (
        id: string,
      ) => {
        if (
          !isEdit ||
          saving
        ) {
          return;
        }

        setDraftAssigneeId(
          String(
            id ?? "",
          ).trim(),
        );
      },
      [
        isEdit,
        saving,
      ],
    );

  const effectiveImageUrls =
    React.useMemo(
      () =>
        isEdit
          ? images.imageUrls
          : viewImageUrls,
      [
        isEdit,
        images.imageUrls,
        viewImageUrls,
      ],
    );

  const [
    mainImageIndex,
    setMainImageIndex,
  ] =
    React.useState(
      0,
    );

  useMainImageIndexGuard({
    imageUrls:
      effectiveImageUrls,
    mainImageIndex,
    setMainImageIndex,
  });

  const onChangePrice =
    React.useCallback(
      (
        index: number,
        price:
          number |
          undefined,
        _row:
          PriceRow,
      ) => {
        if (!isEdit) {
          return;
        }

        setDraftPriceRows(
          (
            previousRows,
          ) =>
            updatePriceRowPrice(
              previousRows,
              index,
              price,
            ),
        );
      },
      [isEdit],
    );

  const onSave =
    React.useCallback(
      async (
        payload?: any,
      ) => {
        const id =
          String(
            listId ?? "",
          ).trim();

        if (!id) {
          setSaveError(
            "invalid_list_id",
          );

          return;
        }

        const nextTitle =
          String(
            payload
              ?.title ??
              "",
          ).trim() ||
          String(
            payload
              ?.listingTitle ??
              "",
          ).trim() ||
          String(
            draftListingTitle ??
              "",
          ).trim();

        const nextDescription =
          payload &&
          payload.description !==
            undefined
            ? String(
                payload
                  .description ??
                  "",
              )
            : String(
                draftDescription ??
                  "",
              );

        const payloadStatus =
          String(
            payload
              ?.status ??
              "",
          ).trim();

        const nextStatus =
          isValidListStatus(
            payloadStatus,
          )
            ? payloadStatus
            : draftStatus;

        const uid =
          String(
            auth.currentUser
              ?.uid ??
              "",
          ).trim() ||
          "system";

        setSaving(true);
        setSaveError("");

        try {
          const result =
            await saveListDetailChanges({
              listId: id,

              currentDTO:
                dto,

              title:
                nextTitle,

              description:
                nextDescription,

              status:
                nextStatus,

              assigneeId:
                String(
                  payload
                    ?.assigneeId ??
                    "",
                ).trim() ||
                String(
                  draftAssigneeId ??
                    "",
                ).trim() ||
                String(
                  dto
                    ?.assigneeId ??
                    "",
                ).trim() ||
                undefined,

              updatedBy:
                uid,

              draftPriceRows:
                Array.isArray(
                  draftPriceRows,
                )
                  ? draftPriceRows
                  : [],

              draftImages:
                Array.isArray(
                  images
                    .draftImages,
                )
                  ? images
                      .draftImages
                  : [],

              mainImageIndex,
            });

          if (
            cancelledRef.current
          ) {
            return;
          }

          revokeDraftBlobUrls(
            images.draftImages,
          );

          setDTO(
            result.dto,
          );

          setIsEdit(
            false,
          );
        } catch (
          caughtError
        ) {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setSaveError(
            String(
              caughtError
                instanceof Error
                ? caughtError
                    .message
                : caughtError,
            ),
          );
        } finally {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setSaving(false);
        }
      },
      [
        listId,
        dto,
        draftStatus,
        draftListingTitle,
        draftDescription,
        draftAssigneeId,
        draftPriceRows,
        images.draftImages,
        mainImageIndex,
        cancelledRef,
      ],
    );

  return {
    loading,
    error,

    saving,
    saveError,

    isEdit,
    onEdit,
    onCancel,
    onSave,

    listingTitle,
    description,

    draftListingTitle,
    setDraftListingTitle,

    draftDescription,
    setDraftDescription,

    status,
    draftStatus,
    onToggleStatus,

    productBrandName,
    productName,

    tokenBrandName,
    tokenName,

    imageUrls:
      effectiveImageUrls,

    onAddImages:
      images.onAddImages,

    onRemoveImageAt:
      images.onRemoveImageAt,

    onClearImages:
      images.onClearImages,

    mainImageIndex,
    setMainImageIndex,

    priceRows:
      viewPriceRows,

    draftPriceRows,
    onChangePrice,

    assigneeId,
    assigneeName,
    draftAssigneeId,
    onSelectAssignee,

    createdByName,
    createdAt,

    updatedByName,
    updatedAt,
  };
}