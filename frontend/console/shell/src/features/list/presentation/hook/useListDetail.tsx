// frontend/console/shell/src/features/list/presentation/hook/useListDetail.tsx

import * as React from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import type { PriceRow } from "../../../inventory/application/listCreateService";
import { useAuthContext } from "../../../../auth/application/AuthContext";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import { useMainImageIndexGuard } from "./internal/useMainImageIndexGuard";
import { useCancelledRef } from "./internal/useCancelledRef";
import { useListImages } from "./useListImages";
import { saveListDetailChanges } from "../../application/listDetail/listDetailSave.usecase";
import { updatePriceRowPrice } from "../../application/listDetail/listDetailMapper";
import { buildListDetailSaveInput } from "../../application/listDetail/buildListDetailSaveInput";
import type { ListDetailSavePayload } from "../../application/listDetail/listDetailSavePayload";
import type { ListStatus } from "../../../../shared/types/list";
import {
  deleteListDetail,
  deriveListDetail,
  loadListDetailDTO,
  resolveListDetailParams,
  type ListDetailDTO,
  type ListDetailRouteParams,
} from "../../application/listDetailService";
import {
  createCompletedListProgress,
  createFailedListProgress,
  createInitialListProgress,
  createPreparingListProgress,
  createSavingListProgress,
  createUploadingListProgress,
  isListProgressVisible,
  type ListProgress,
} from "../modal/listProgress";

export type { DraftImage } from "./useListImages";

export type UseListDetailResult = {
  loading: boolean;
  error: string;
  saving: boolean;
  saveError: string;
  deleting: boolean;
  deleteError: string;

  isEdit: boolean;

  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onDelete: () => Promise<void>;
  onSave: (
    payload?: ListDetailSavePayload,
  ) => Promise<void>;

  progress: ListProgress;
  progressOpen: boolean;
  onCloseProgress: () => void;

  readableId: string;
  listingTitle: string;
  description: string;

  draftListingTitle: string;
  setDraftListingTitle:
    React.Dispatch<
      React.SetStateAction<string>
    >;

  draftDescription: string;
  setDraftDescription:
    React.Dispatch<
      React.SetStateAction<string>
    >;

  status: ListStatus | "";
  draftStatus: ListStatus;

  /**
   * 編集モード内でdraftのstatusを変更する。
   */
  onToggleStatus: (
    next: ListStatus,
  ) => void;

  /**
   * 閲覧モードのヘッダーからstatusを即時保存する。
   */
  onChangeStatus: (
    next: ListStatus,
  ) => void;

  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;

  totalOrderCount: number;
  totalSalesAmount: number;

  imageUrls: string[];

  onAddImages: (
    files: FileList | null,
  ) => void;

  onRemoveImageAt: (
    index: number,
  ) => void;

  onClearImages: () => void;

  mainImageIndex: number;

  setMainImageIndex:
    React.Dispatch<
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
  draftAssigneeName: string;

  assigneeCandidates: {
    id: string;
    name: string;
  }[];

  loadingMembers: boolean;

  onSelectAssignee: (
    id: string,
  ) => void;

  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;
};

function clonePriceRows(
  rows: readonly PriceRow[],
): PriceRow[] {
  return rows.map(
    (row) => ({
      ...row,
    }),
  );
}

function errorMessageFromUnknown(
  error: unknown,
): string {
  return String(
    error instanceof Error
      ? error.message
      : error,
  );
}

export function useListDetail():
  UseListDetailResult {
  const params =
    useParams<ListDetailRouteParams>();

  const navigate =
    useNavigate();

  const { user } =
    useAuthContext();

  const resolved = React.useMemo(() =>
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
      ListDetailDTO | null
    >(null);

  const [
    loading,
    setLoading,
  ] =
    React.useState(false);

  const [
    error,
    setError,
  ] =
    React.useState("");

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
              caughtError instanceof Error
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

  React.useEffect(
    () => {
      void reload();
    },
    [
      reload,
    ],
  );

  const derived =
    React.useMemo(
      () =>
        dto
          ? deriveListDetail(
              dto,
            )
          : null,
      [
        dto,
      ],
    );

  const readableId =
    derived?.readableId ??
    "";

  const listingTitle =
    derived?.title ??
    "";

  const description =
    derived?.description ??
    "";

  const status =
    derived?.status ??
    "";

  const productBrandName =
    derived?.productBrandName ??
    "";

  const productName =
    derived?.productName ??
    "";

  const tokenBrandName =
    derived?.tokenBrandName ??
    "";

  const tokenName =
    derived?.tokenName ??
    "";

  const totalOrderCount =
    derived?.totalOrderCount ??
    0;

  const totalSalesAmount =
    derived?.totalSalesAmount ??
    0;

  const assigneeId =
    derived?.assigneeId ??
    "";

  const assigneeName =
    derived?.assigneeName ??
    "";

  const createdByName =
    derived?.createdByName ??
    "";

  const createdAt =
    derived?.createdAtLabel ??
    "";

  const updatedByName =
    derived?.updatedByName ??
    "";

  const updatedAt =
    derived?.updatedAtLabel ??
    "";

  const primaryImageId =
    derived?.primaryImageId;

  const {
    assigneeId:
      draftAssigneeId,

    assigneeName:
      draftAssigneeName,

    assigneeCandidates,

    loadingMembers,

    handleSelectAssignee,

    resetAssignee,
  } =
    useAssigneeSelection({
      initialAssigneeId:
        assigneeId ||
        null,

      initialAssigneeName:
        assigneeName ||
        null,

      defaultToCurrentMember:
        false,
    });

  const viewImages =
    React.useMemo(
      () =>
        derived?.images ??
        [],
      [
        derived,
      ],
    );

  const viewImageUrls =
    React.useMemo(
      () =>
        viewImages.map(
          (image) =>
            image.url,
        ),
      [
        viewImages,
      ],
    );

  const viewPriceRows =
    React.useMemo<
      PriceRow[]
    >(
      () =>
        derived?.priceRows ??
        [],
      [
        derived,
      ],
    );

  const viewPrimaryImageIndex =
    React.useMemo(
      () => {
        if (
          !primaryImageId
        ) {
          return 0;
        }

        const index =
          viewImages.findIndex(
            (image) =>
              image.id ===
              primaryImageId,
          );

        return index >= 0
          ? index
          : 0;
      },
      [
        primaryImageId,
        viewImages,
      ],
    );

  const [
    isEdit,
    setIsEdit,
  ] =
    React.useState(false);

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
      "suspended",
    );

  const [
    saving,
    setSaving,
  ] =
    React.useState(false);

  const [
    saveError,
    setSaveError,
  ] =
    React.useState("");

  const [
    deleting,
    setDeleting,
  ] =
    React.useState(false);

  const [
    deleteError,
    setDeleteError,
  ] =
    React.useState("");

  const [
    progress,
    setProgress,
  ] =
    React.useState<
      ListProgress
    >(
      createInitialListProgress,
    );

  const progressOpen =
    isListProgressVisible(
      progress,
    );

  const [
    mainImageIndex,
    setMainImageIndex,
  ] =
    React.useState(
      viewPrimaryImageIndex,
    );

  const images =
    useListImages({
      isEdit,
      saving,
      initialImages:
        viewImages,
    });

  React.useEffect(
    () => {
      if (
        !progress.isBlockingNavigation
      ) {
        return;
      }

      const handleBeforeUnload =
        (
          event:
            BeforeUnloadEvent,
        ) => {
          event.preventDefault();

          event.returnValue =
            "";
        };

      window.addEventListener(
        "beforeunload",
        handleBeforeUnload,
      );

      return () => {
        window.removeEventListener(
          "beforeunload",
          handleBeforeUnload,
        );
      };
    },
    [
      progress.isBlockingNavigation,
    ],
  );

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

        if (
          status
        ) {
          setDraftStatus(
            status,
          );
        }

        resetAssignee();
      },
      [
        listingTitle,
        description,
        viewPriceRows,
        status,
        resetAssignee,
      ],
    );

  React.useEffect(
    () => {
      if (
        isEdit
      ) {
        return;
      }

      resetDraftFromView();
    },
    [
      isEdit,
      resetDraftFromView,
    ],
  );

  React.useEffect(
    () => {
      if (
        isEdit
      ) {
        return;
      }

      setMainImageIndex(
        viewPrimaryImageIndex,
      );
    },
    [
      isEdit,
      viewPrimaryImageIndex,
    ],
  );

  const onBack =
    React.useCallback(
      () => {
        if (
          progress.isBlockingNavigation ||
          deleting
        ) {
          return;
        }

        navigate(
          "/list",
        );
      },
      [
        progress.isBlockingNavigation,
        deleting,
        navigate,
      ],
    );

  const onCloseProgress =
    React.useCallback(
      () => {
        if (
          progress.isBlockingNavigation
        ) {
          return;
        }

        setProgress(
          createInitialListProgress(),
        );
      },
      [
        progress.isBlockingNavigation,
      ],
    );

  const onEdit =
    React.useCallback(
      () => {
        if (
          deleting ||
          progress.isBlockingNavigation
        ) {
          return;
        }

        resetDraftFromView();

        setMainImageIndex(
          viewPrimaryImageIndex,
        );

        setSaveError(
          "",
        );

        setDeleteError(
          "",
        );

        setIsEdit(
          true,
        );
      },
      [
        deleting,
        progress.isBlockingNavigation,
        resetDraftFromView,
        viewPrimaryImageIndex,
      ],
    );

  const onCancel =
    React.useCallback(
      () => {
        if (
          saving ||
          deleting ||
          progress.isBlockingNavigation
        ) {
          return;
        }

        images.releaseDraftBlobUrls();

        resetDraftFromView();

        setMainImageIndex(
          viewPrimaryImageIndex,
        );

        setSaveError(
          "",
        );

        setDeleteError(
          "",
        );

        setIsEdit(
          false,
        );
      },
      [
        saving,
        deleting,
        progress.isBlockingNavigation,
        images.releaseDraftBlobUrls,
        resetDraftFromView,
        viewPrimaryImageIndex,
      ],
    );

  const onDelete =
    React.useCallback(
      async () => {
        const id =
          String(
            listId ?? "",
          ).trim();

        if (
          !id
        ) {
          setDeleteError(
            "invalid_list_id",
          );

          return;
        }

        if (
          saving ||
          deleting ||
          progress.isBlockingNavigation
        ) {
          return;
        }

        const confirmed =
          window.confirm(
            "この出品を削除しますか？削除後は元に戻せません。",
          );

        if (
          !confirmed
        ) {
          return;
        }

        setDeleting(
          true,
        );

        setDeleteError(
          "",
        );

        try {
          await deleteListDetail(
            id,
          );

          if (
            cancelledRef.current
          ) {
            return;
          }

          images.releaseDraftBlobUrls();

          navigate(
            "/list",
            {
              replace: true,
            },
          );
        } catch (
          caughtError
        ) {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setDeleteError(
            String(
              caughtError instanceof Error
                ? caughtError.message
                : caughtError,
            ),
          );
        } finally {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setDeleting(
            false,
          );
        }
      },
      [
        listId,
        saving,
        deleting,
        progress.isBlockingNavigation,
        cancelledRef,
        images.releaseDraftBlobUrls,
        navigate,
      ],
    );

  /**
   * 編集画面内でstatus draftを変更するためのhandler。
   *
   * ListDetailのヘッダーは閲覧モードに表示されるため、
   * ヘッダーからの即時更新にはonChangeStatusを使用する。
   */
  const onToggleStatus =
    React.useCallback(
      (
        next:
          ListStatus,
      ) => {
        if (
          !isEdit ||
          saving ||
          deleting
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
        deleting,
      ],
    );

  const onSelectAssignee =
    React.useCallback(
      (
        id:
          string,
      ) => {
        if (
          !isEdit ||
          saving ||
          deleting
        ) {
          return;
        }

        handleSelectAssignee(
          id,
        );
      },
      [
        isEdit,
        saving,
        deleting,
        handleSelectAssignee,
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

  useMainImageIndexGuard({
    imageUrls:
      effectiveImageUrls,

    mainImageIndex,

    setMainImageIndex,
  });

  const onChangePrice =
    React.useCallback(
      (
        index:
          number,

        price:
          number |
          undefined,

        _row:
          PriceRow,
      ) => {
        if (
          !isEdit ||
          saving ||
          deleting
        ) {
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
      [
        isEdit,
        saving,
        deleting,
      ],
    );

  const onSave =
    React.useCallback(
      async (
        payload?:
          ListDetailSavePayload,
      ) => {
        const id =
          String(
            listId ?? "",
          ).trim();

        if (
          !id
        ) {
          setSaveError(
            "invalid_list_id",
          );

          return;
        }

        if (
          !dto
        ) {
          setSaveError(
            "list_detail_not_loaded",
          );

          return;
        }

        if (
          saving ||
          deleting ||
          progress.isBlockingNavigation
        ) {
          return;
        }

        /*
         * 通常はdraftAssigneeIdが最新DTOへ同期済みだが、
         * DTO読み込み直後にヘッダーのstatusボタンが押された場合でも
         * current assigneeを失わないようview側のassigneeIdへfallbackする。
         */
        const effectiveDraftAssigneeId =
          draftAssigneeId ||
          assigneeId;

        if (
          !effectiveDraftAssigneeId
        ) {
          setSaveError(
            "assignee_required",
          );

          return;
        }

        let transferredBytes =
          0;

        let totalBytes =
          0;

        let completedUploadCount =
          0;

        let expectedUploadCount =
          0;

        setSaving(
          true,
        );

        setSaveError(
          "",
        );

        setDeleteError(
          "",
        );

        try {
          const saveInput =
            buildListDetailSaveInput({
              payload,

              draftListingTitle,

              draftDescription,

              draftStatus,

              draftAssigneeId:
                effectiveDraftAssigneeId,

              currentUserUid:
                user?.uid,
            });

          const result =
            await saveListDetailChanges({
              listId:
                id,

              currentDTO:
                dto,

              title:
                saveInput.title,

              description:
                saveInput.description,

              status:
                saveInput.status,

              assigneeId:
                saveInput.assigneeId,

              updatedBy:
                saveInput.updatedBy,

              draftPriceRows,

              draftImages:
                images.draftImages,

              mainImageIndex,

              progressHandlers: {
                onPreparing:
                  () => {
                    setProgress(
                      createPreparingListProgress({
                        title:
                          "保存準備中",

                        message:
                          "出品情報の保存準備をしています。",
                      }),
                    );
                  },

                onImageProgress:
                  (
                    imageProgress,
                  ) => {
                    transferredBytes =
                      imageProgress.transferredBytes;

                    totalBytes =
                      imageProgress.totalBytes;

                    completedUploadCount =
                      imageProgress.completedUploadCount;

                    expectedUploadCount =
                      imageProgress.expectedUploadCount;

                    setProgress(
                      createUploadingListProgress({
                        fileName:
                          imageProgress.fileName,

                        transferredBytes:
                          imageProgress.transferredBytes,

                        totalBytes:
                          imageProgress.totalBytes,

                        completedUploadCount:
                          imageProgress.completedUploadCount,

                        expectedUploadCount:
                          imageProgress.expectedUploadCount,
                      }),
                    );
                  },

                onSaving:
                  (
                    savingProgress,
                  ) => {
                    transferredBytes =
                      savingProgress.transferredBytes;

                    totalBytes =
                      savingProgress.totalBytes;

                    completedUploadCount =
                      savingProgress.completedUploadCount;

                    expectedUploadCount =
                      savingProgress.expectedUploadCount;

                    setProgress(
                      createSavingListProgress({
                        transferredBytes:
                          savingProgress.transferredBytes,

                        totalBytes:
                          savingProgress.totalBytes,

                        completedUploadCount:
                          savingProgress.completedUploadCount,

                        expectedUploadCount:
                          savingProgress.expectedUploadCount,

                        title:
                          "出品を保存中",

                        message:
                          "画像情報と出品情報を保存しています。この画面を閉じたり移動したりしないでください。",
                      }),
                    );
                  },
              },
            });

          if (
            cancelledRef.current
          ) {
            return;
          }

          images.releaseDraftBlobUrls();

          setDTO(
            result.dto,
          );

          setIsEdit(
            false,
          );

          setProgress(
            createCompletedListProgress({
              transferredBytes,

              totalBytes,

              completedUploadCount,

              expectedUploadCount,

              title:
                "保存が完了しました",

              message:
                "出品情報の保存が完了しました。",
            }),
          );
        } catch (
          caughtError
        ) {
          if (
            cancelledRef.current
          ) {
            return;
          }

          const message =
            errorMessageFromUnknown(
              caughtError,
            );

          setSaveError(
            message,
          );

          setProgress(
            createFailedListProgress(
              message,
              {
                title:
                  "保存に失敗しました",

                message:
                  "出品情報の保存中にエラーが発生しました。",
              },
            ),
          );
        } finally {
          if (
            cancelledRef.current
          ) {
            return;
          }

          setSaving(
            false,
          );
        }
      },
      [
        listId,
        dto,
        user?.uid,
        draftStatus,
        draftListingTitle,
        draftDescription,
        draftAssigneeId,
        assigneeId,
        draftPriceRows,
        images.draftImages,
        images.releaseDraftBlobUrls,
        mainImageIndex,
        saving,
        deleting,
        progress.isBlockingNavigation,
        cancelledRef,
      ],
    );

  /**
   * ListDetailヘッダーの「出品 / 保留」から呼び出す。
   *
   * 閲覧モードでは編集画面へ遷移せず、
   * statusだけを既存save-operation経由で即時保存する。
   */
  const onChangeStatus =
    React.useCallback(
      (
        next:
          ListStatus,
      ) => {
        if (
          isEdit ||
          loading ||
          saving ||
          deleting ||
          progress.isBlockingNavigation ||
          !dto ||
          !status
        ) {
          return;
        }

        /*
         * 現在と同じstatusの場合はAPIを呼ばない。
         * ListStatusHeaderActions側にも同じguardがあるが、
         * hook側でも防御しておく。
         */
        if (
          status === next
        ) {
          return;
        }

        void onSave({
          status:
            next,
        });
      },
      [
        isEdit,
        loading,
        saving,
        deleting,
        progress.isBlockingNavigation,
        dto,
        status,
        onSave,
      ],
    );

  return {
    loading,
    error,
    saving,
    saveError,
    deleting,
    deleteError,

    isEdit,

    onBack,
    onEdit,
    onCancel,
    onDelete,
    onSave,

    progress,
    progressOpen,
    onCloseProgress,

    readableId,
    listingTitle,
    description,

    draftListingTitle,
    setDraftListingTitle,

    draftDescription,
    setDraftDescription,

    status,
    draftStatus,
    onToggleStatus,
    onChangeStatus,

    productBrandName,
    productName,
    tokenBrandName,
    tokenName,

    totalOrderCount,
    totalSalesAmount,

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
    draftAssigneeName,

    assigneeCandidates,
    loadingMembers,
    onSelectAssignee,

    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
  };
}