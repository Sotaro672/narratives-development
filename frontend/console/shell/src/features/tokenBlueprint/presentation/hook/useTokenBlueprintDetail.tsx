// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintDetail.tsx

import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import type { ContentFile, TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import {
  deleteTokenBlueprintDetail,
  fetchTokenBlueprintDetail,
  updateTokenBlueprintFromCard,
} from "../../application/tokenBlueprintDetailService";
import { createTokenBlueprintContentId } from "../../application/tokenBlueprintContentService";
import { patchTokenBlueprintContentFiles } from "../../infrastructure/repository/tokenBlueprintRepositoryHTTP";
import { uploadTokenBlueprintContentToFirebaseStorage } from "../../infrastructure/storage/tokenBlueprintAssetStorage";
import {
  createCompletedTokenBlueprintProgress,
  createFailedTokenBlueprintProgress,
  createInitialTokenBlueprintProgress,
  createPreparingTokenBlueprintProgress,
  createSavingTokenBlueprintProgress,
  createUploadingTokenBlueprintProgress,
  isTokenBlueprintProgressVisible,
  type TokenBlueprintProgress,
  type TokenBlueprintUploadProgressInput,
} from "../model/tokenBlueprintProgress";
import { useTokenBlueprintCard } from "./useTokenBlueprintCard";

type TokenBlueprintCardHookResult = ReturnType<typeof useTokenBlueprintCard>;

type UseTokenBlueprintDetailVM = {
  blueprint: TokenBlueprint | null;
  title: string;
  assigneeId: string;
  assigneeName: string;
  assigneeCandidates: {
    id: string;
    name: string;
  }[];
  loadingMembers: boolean;
  minted: boolean;
  createdByName: string;
  createdAt: string;
  updatedByName: string;
  updatedAt: string;
  tokenContents: ContentFile[];
  cardVm: TokenBlueprintCardHookResult["vm"];
  isEditMode: boolean;
  isUploadingContents: boolean;
  progress: TokenBlueprintProgress;
  progressOpen: boolean;
};

type UseTokenBlueprintDetailHandlers = {
  onBack: () => void;
  onEdit: () => void;
  onCancel: () => void;
  onSave: () => Promise<void>;
  onDelete: () => Promise<void>;
  onSelectAssignee: (id: string) => void;
  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onCloseProgress: () => void;
  cardHandlers: TokenBlueprintCardHookResult["handlers"];
  onTokenContentsFilesSelected: (files: File[]) => Promise<void>;
  onDeleteTokenContent: (item: ContentFile, index: number) => Promise<void>;
};

export type UseTokenBlueprintDetailResult = {
  vm: UseTokenBlueprintDetailVM;
  handlers: UseTokenBlueprintDetailHandlers;
};

type ExistingTokenBlueprintContentsProgressHandlers = {
  onUploadProgress?: (progress: TokenBlueprintUploadProgressInput) => void;
  onSaving?: (progress: {
    transferredBytes: number;
    totalBytes: number;
    completedUploadCount: number;
    expectedUploadCount: number;
  }) => void;
};

function errorMessageFromUnknown(
  error: unknown,
): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "トークン設計の保存に失敗しました。";
}

async function uploadAndAppendExistingTokenBlueprintContents(params: {
  companyId: string;
  tokenBlueprintId: string;
  actorId: string;
  files: File[];
  existingContentFiles: ContentFile[];
  progressHandlers?: ExistingTokenBlueprintContentsProgressHandlers;
}): Promise<TokenBlueprint> {
  let currentContentFiles = [...params.existingContentFiles];
  let updatedBlueprint: TokenBlueprint | null = null;

  const totalBytes = params.files.reduce(
    (total, file) => total + file.size,
    0,
  );

  let completedBytes = 0;

  for (const [index, file] of params.files.entries()) {
    const contentId = createTokenBlueprintContentId();

    const uploaded = await uploadTokenBlueprintContentToFirebaseStorage({
      companyId: params.companyId,
      tokenBlueprintId: params.tokenBlueprintId,
      contentId,
      file,
      onProgress: (progress) => {
        params.progressHandlers?.onUploadProgress?.({
          target: "content",
          fileName: file.name,
          transferredBytes:
            completedBytes +
            progress.transferredBytes,
          totalBytes,
          completedUploadCount: index,
          expectedUploadCount: params.files.length,
        });
      },
    });

    completedBytes += file.size;

    const nowIso = new Date().toISOString();

    const contentFile: ContentFile = {
      id: contentId,
      name: uploaded.fileName,
      type: uploaded.kind,
      contentType: uploaded.contentType,
      objectPath: uploaded.objectPath,
      url: uploaded.downloadUrl,
      size: uploaded.size,
      isPublic: false,
      createdAt: nowIso,
      createdBy: params.actorId,
      updatedAt: nowIso,
      updatedBy: params.actorId,
    };

    currentContentFiles = [
      ...currentContentFiles,
      contentFile,
    ];

    params.progressHandlers?.onSaving?.({
      transferredBytes: completedBytes,
      totalBytes,
      completedUploadCount: index + 1,
      expectedUploadCount: params.files.length,
    });

    updatedBlueprint = await patchTokenBlueprintContentFiles({
      tokenBlueprintId: params.tokenBlueprintId,
      contentFiles: currentContentFiles,
    });

    currentContentFiles = updatedBlueprint.contentFiles;
  }

  if (!updatedBlueprint) {
    throw new Error("files must contain at least one file");
  }

  return updatedBlueprint;
}

export function useTokenBlueprintDetail(): UseTokenBlueprintDetailResult {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();
  const { currentMember } = useAuthContext();
  const memberId = currentMember?.id ?? "";

  const [blueprint, setBlueprint] = useState<TokenBlueprint | null>(null);
  const [loading, setLoading] = useState(false);
  const [progress, setProgress] = useState<TokenBlueprintProgress>(
    createInitialTokenBlueprintProgress,
  );

  const {
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    handleSelectAssignee,
    resetAssignee,
  } = useAssigneeSelection({
    initialAssigneeId: blueprint?.assigneeId ?? null,
    initialAssigneeName: blueprint?.assigneeName ?? null,
    defaultToCurrentMember: false,
  });

  useEffect(() => {
    const id = tokenBlueprintId;

    if (!id) {
      return;
    }

    let cancelled = false;

    const load = async (): Promise<void> => {
      try {
        setLoading(true);

        const result = await fetchTokenBlueprintDetail(id);

        if (cancelled) {
          return;
        }

        setBlueprint(result);
      } catch {
        if (!cancelled) {
          navigate("/tokenBlueprint", { replace: true });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, navigate]);

  useEffect(() => {
    if (!progress.isBlockingNavigation) {
      return;
    }

    const handleBeforeUnload = (
      event: BeforeUnloadEvent,
    ) => {
      event.preventDefault();
      event.returnValue = "";
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
  }, [progress.isBlockingNavigation]);

  const minted = blueprint?.minted ?? false;
  const createdByName = blueprint?.createdByName ?? "";
  const updatedByName = blueprint?.updatedByName ?? "";
  const createdAt = safeDateTimeLabelJa(blueprint?.createdAt ?? "", "");
  const updatedAt = safeDateTimeLabelJa(blueprint?.updatedAt ?? "", "");
  const initialIconUrl = blueprint?.iconUrl ?? undefined;
  const tokenContents: ContentFile[] = blueprint?.contentFiles ?? [];

  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: blueprint ?? undefined,
    initialBurnAt: "",
    initialIconUrl,
    initialEditMode: false,
  });

  const isEditMode = cardVm.isEditMode;
  const progressOpen = isTokenBlueprintProgressVisible(progress);
  const isUploadingContents =
    progress.phase === "uploading" &&
    progress.currentUploadTarget === "content";

  const handleBack = useCallback(() => {
    if (progress.isBlockingNavigation) {
      return;
    }

    navigate("/tokenBlueprint", { replace: true });
  }, [progress.isBlockingNavigation, navigate]);

  const handleEdit = useCallback(() => {
    if (progress.isBlockingNavigation) {
      return;
    }

    cardHandlers.setEditMode?.(true);
  }, [progress.isBlockingNavigation, cardHandlers]);

  const handleCancel = useCallback(() => {
    if (progress.isBlockingNavigation) {
      return;
    }

    cardHandlers.reset?.();
    cardHandlers.setEditMode?.(false);
    resetAssignee();
  }, [
    progress.isBlockingNavigation,
    cardHandlers,
    resetAssignee,
  ]);

  const handleCloseProgress = useCallback(() => {
    if (progress.isBlockingNavigation) {
      return;
    }

    setProgress(
      createInitialTokenBlueprintProgress(),
    );
  }, [progress.isBlockingNavigation]);

  const handleSave = useCallback(async (): Promise<void> => {
    if (
      loading ||
      progress.isBlockingNavigation ||
      !blueprint
    ) {
      return;
    }

    if (!assigneeId) {
      window.alert("担当者を選択してください。");
      return;
    }

    const iconFile = cardVm.iconFile ?? null;
    const totalBytes = iconFile?.size ?? 0;

    let transferredBytes = 0;

    try {
      setLoading(true);

      setProgress(
        createPreparingTokenBlueprintProgress({
          title: "保存準備中",
          message: "トークン設計の変更内容を確認しています。",
        }),
      );

      const sourceBlueprint: TokenBlueprint = {
        ...blueprint,
        assigneeId,
        assigneeName,
      };

      const updated = await updateTokenBlueprintFromCard(
        sourceBlueprint,
        cardVm,
        {
          onSaving: () => {
            setProgress(
              createSavingTokenBlueprintProgress({
                transferredBytes,
                totalBytes,
                completedUploadCount:
                  iconFile &&
                  totalBytes > 0 &&
                  transferredBytes >= totalBytes
                    ? 1
                    : 0,
                expectedUploadCount:
                  iconFile
                    ? 1
                    : 0,
              }),
            );
          },

          onIconProgress: (uploadProgress) => {
            transferredBytes =
              uploadProgress.transferredBytes;

            setProgress(
              createUploadingTokenBlueprintProgress({
                target: "icon",
                fileName: iconFile?.name ?? "",
                transferredBytes:
                  uploadProgress.transferredBytes,
                totalBytes:
                  uploadProgress.totalBytes,
                completedUploadCount:
                  uploadProgress.percentage >= 100
                    ? 1
                    : 0,
                expectedUploadCount: 1,
              }),
            );
          },
        },
      );

      setBlueprint(updated);
      cardHandlers.setEditMode?.(false);

      setProgress(
        createCompletedTokenBlueprintProgress({
          transferredBytes: totalBytes,
          totalBytes,
          completedUploadCount:
            iconFile
              ? 1
              : 0,
          expectedUploadCount:
            iconFile
              ? 1
              : 0,
        }),
      );
    } catch (error) {
      setProgress(
        createFailedTokenBlueprintProgress(
          errorMessageFromUnknown(error),
        ),
      );
    } finally {
      setLoading(false);
    }
  }, [
    loading,
    progress.isBlockingNavigation,
    blueprint,
    assigneeId,
    assigneeName,
    cardVm,
    cardHandlers,
  ]);

  const handleDelete = useCallback(async (): Promise<void> => {
    if (
      loading ||
      progress.isBlockingNavigation ||
      !blueprint ||
      blueprint.minted
    ) {
      return;
    }

    try {
      setLoading(true);
      await deleteTokenBlueprintDetail(blueprint.id);
      navigate("/tokenBlueprint", { replace: true });
    } catch {
      setLoading(false);
    }
  }, [
    loading,
    progress.isBlockingNavigation,
    blueprint,
    navigate,
  ]);

  const handleEditAssignee = useCallback(() => {
    // AdminCard互換イベント。
  }, []);

  const handleClickAssignee = useCallback(() => {
    // AdminCard互換イベント。
  }, []);

  const onTokenContentsFilesSelected = useCallback(
    async (files: File[]): Promise<void> => {
      const id = tokenBlueprintId;

      if (
        progress.isBlockingNavigation ||
        !id ||
        !blueprint ||
        files.length === 0
      ) {
        return;
      }

      if (!blueprint.companyId) {
        throw new Error("companyId is required");
      }

      if (!memberId) {
        throw new Error("memberId is required");
      }

      const totalBytes = files.reduce(
        (total, file) => total + file.size,
        0,
      );

      try {
        setProgress(
          createPreparingTokenBlueprintProgress({
            title: "アップロード準備中",
            message: "追加するコンテンツを準備しています。",
          }),
        );

        const updated = await uploadAndAppendExistingTokenBlueprintContents({
          companyId: blueprint.companyId,
          tokenBlueprintId: id,
          actorId: memberId,
          files,
          existingContentFiles: blueprint.contentFiles,
          progressHandlers: {
            onUploadProgress: (uploadProgress) => {
              setProgress(
                createUploadingTokenBlueprintProgress(
                  uploadProgress,
                ),
              );
            },

            onSaving: (savingProgress) => {
              setProgress(
                createSavingTokenBlueprintProgress({
                  ...savingProgress,
                  title: "コンテンツを保存中",
                  message: "転送したコンテンツ情報を保存しています。",
                }),
              );
            },
          },
        });

        setBlueprint(updated);

        setProgress(
          createCompletedTokenBlueprintProgress({
            transferredBytes: totalBytes,
            totalBytes,
            completedUploadCount: files.length,
            expectedUploadCount: files.length,
            title: "コンテンツの追加が完了しました",
            message: "選択したコンテンツの保存が完了しました。",
          }),
        );
      } catch (error) {
        setProgress(
          createFailedTokenBlueprintProgress(
            errorMessageFromUnknown(error),
            {
              title: "コンテンツを追加できませんでした",
              message: "コンテンツの保存中にエラーが発生しました。",
            },
          ),
        );
      }
    },
    [
      tokenBlueprintId,
      blueprint,
      memberId,
      progress.isBlockingNavigation,
    ],
  );

  const onDeleteTokenContent = useCallback(
    async (
      item: ContentFile,
      _index: number,
    ): Promise<void> => {
      const id = tokenBlueprintId;

      if (
        progress.isBlockingNavigation ||
        !id ||
        !blueprint
      ) {
        return;
      }

      const nextContentFiles = blueprint.contentFiles.filter(
        (contentFile) => contentFile.id !== item.id,
      );

      try {
        setProgress(
          createSavingTokenBlueprintProgress({
            title: "コンテンツを削除中",
            message: "トークン設計からコンテンツを削除しています。",
          }),
        );

        const updated = await patchTokenBlueprintContentFiles({
          tokenBlueprintId: id,
          contentFiles: nextContentFiles,
        });

        setBlueprint(updated);

        setProgress(
          createCompletedTokenBlueprintProgress({
            title: "コンテンツを削除しました",
            message: "コンテンツの削除が完了しました。",
          }),
        );
      } catch (error) {
        setProgress(
          createFailedTokenBlueprintProgress(
            errorMessageFromUnknown(error),
            {
              title: "コンテンツを削除できませんでした",
              message: "コンテンツの削除中にエラーが発生しました。",
            },
          ),
        );
      }
    },
    [
      tokenBlueprintId,
      blueprint,
      progress.isBlockingNavigation,
    ],
  );

  const vm: UseTokenBlueprintDetailVM = {
    blueprint,
    title: "トークン設計",
    assigneeId,
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    minted,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    tokenContents,
    cardVm,
    isEditMode,
    isUploadingContents,
    progress,
    progressOpen,
  };

  const handlers: UseTokenBlueprintDetailHandlers = {
    onBack: handleBack,
    onEdit: handleEdit,
    onCancel: handleCancel,
    onSave: handleSave,
    onDelete: handleDelete,
    onSelectAssignee: handleSelectAssignee,
    onEditAssignee: handleEditAssignee,
    onClickAssignee: handleClickAssignee,
    onCloseProgress: handleCloseProgress,
    cardHandlers,
    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  };

  return { vm, handlers };
}