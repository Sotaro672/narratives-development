// frontend/console/shell/src/features/announcement/presentation/hook/useAnnouncementCreatePage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import type { AnnouncementInputPayload } from "../../application/announcement_input";

import {
  createEmptyAnnouncementCreateVM,
  fetchAnnouncementCreateVM,
  normalizeAnnouncementCreateLocationState,
  saveAnnouncement,
  sendAnnouncement,
  type AnnouncementCreateVM,
  type AnnouncementCreateProgressHandlers,
  type AnnouncementOwnerVM,
} from "../../application/announcement_create_service";

import {
  createCompletedAnnouncementCreateProgress,
  createFailedAnnouncementCreateProgress,
  createInitialAnnouncementCreateProgress,
  createPreparingAnnouncementCreateProgress,
  createSavingAnnouncementCreateProgress,
  createUploadingAnnouncementCreateProgress,
  isAnnouncementCreateProgressVisible,
  type AnnouncementCreateProgress,
} from "../model/announcementCreateProgress";

export type { AnnouncementOwnerVM };

export type SubmitAnnouncementParams = {
  payload: AnnouncementInputPayload;
  targetAvatarIds: string[];
};

type CompletedAction =
  | {
      type: "save";
    }
  | {
      type: "send";
      announcementId: string;
    }
  | null;

type UseAnnouncementCreatePageState = {
  isSaving: boolean;
  isSending: boolean;
  progress: AnnouncementCreateProgress;
  progressOpen: boolean;
};

type UseAnnouncementCreatePageHandlers = {
  onBack: () => void;
  onSaveAnnouncement: (params: SubmitAnnouncementParams) => Promise<void>;
  onSendAnnouncement: (params: SubmitAnnouncementParams) => Promise<string>;
  onCloseProgress: () => void;
};

export type UseAnnouncementCreatePageResult = {
  vm: AnnouncementCreateVM;
  state: UseAnnouncementCreatePageState;
  handlers: UseAnnouncementCreatePageHandlers;
};

type ProgressSnapshot = {
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error
    ? error.message
    : String(error);
}

function createInitialProgressSnapshot(): ProgressSnapshot {
  return {
    transferredBytes: 0,
    totalBytes: 0,
    completedUploadCount: 0,
    expectedUploadCount: 0,
  };
}

export function useAnnouncementCreatePage(): UseAnnouncementCreatePageResult {
  const navigate = useNavigate();
  const location = useLocation();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [vm, setVm] = useState<AnnouncementCreateVM>(() =>
    createEmptyAnnouncementCreateVM(),
  );

  const [isSaving, setIsSaving] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [progress, setProgress] = useState<AnnouncementCreateProgress>(
    createInitialAnnouncementCreateProgress,
  );
  const [completedAction, setCompletedAction] =
    useState<CompletedAction>(null);

  const locationState = useMemo(
    () => normalizeAnnouncementCreateLocationState(location.state),
    [location.state],
  );

  const progressOpen =
    isAnnouncementCreateProgressVisible(progress);

  const isBusy =
    isSaving ||
    isSending;

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const nextVm = await fetchAnnouncementCreateVM(
          tokenBlueprintId,
          locationState,
        );

        if (cancelled) {
          return;
        }

        setVm(nextVm);
      } catch {
        if (!cancelled) {
          navigate("/sales/create", {
            replace: true,
          });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [
    tokenBlueprintId,
    locationState,
    navigate,
  ]);

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

  const handleBack = useCallback(() => {
    if (isBusy || progress.isBlockingNavigation) {
      return;
    }

    navigate("/sales/create", {
      replace: true,
    });
  }, [
    isBusy,
    progress.isBlockingNavigation,
    navigate,
  ]);

  const resolveCreatedBy = useCallback(() => {
    return (
      vm.updatedById ||
      vm.createdById ||
      vm.updatedByName ||
      vm.createdByName ||
      "system"
    );
  }, [
    vm.createdById,
    vm.createdByName,
    vm.updatedById,
    vm.updatedByName,
  ]);

  const createProgressHandlers = useCallback(
    (
      snapshot: ProgressSnapshot,
      savingTitle: string,
      savingMessage: string,
    ): AnnouncementCreateProgressHandlers => ({
      onPreparing: () => {
        setProgress(
          createPreparingAnnouncementCreateProgress({
            title: "処理を準備中",
            message: "告知情報の保存準備をしています。",
          }),
        );
      },

      onImageProgress: (uploadProgress) => {
        snapshot.transferredBytes =
          uploadProgress.transferredBytes;
        snapshot.totalBytes =
          uploadProgress.totalBytes;
        snapshot.completedUploadCount =
          uploadProgress.completedUploadCount;
        snapshot.expectedUploadCount =
          uploadProgress.expectedUploadCount;

        setProgress(
          createUploadingAnnouncementCreateProgress({
            fileName: uploadProgress.fileName,
            transferredBytes:
              uploadProgress.transferredBytes,
            totalBytes:
              uploadProgress.totalBytes,
            completedUploadCount:
              uploadProgress.completedUploadCount,
            expectedUploadCount:
              uploadProgress.expectedUploadCount,
            title: "画像を転送中",
            message:
              "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。",
          }),
        );
      },

      onSaving: (savingProgress) => {
        snapshot.transferredBytes =
          savingProgress.transferredBytes;
        snapshot.totalBytes =
          savingProgress.totalBytes;
        snapshot.completedUploadCount =
          savingProgress.completedUploadCount;
        snapshot.expectedUploadCount =
          savingProgress.expectedUploadCount;

        setProgress(
          createSavingAnnouncementCreateProgress({
            transferredBytes:
              savingProgress.transferredBytes,
            totalBytes:
              savingProgress.totalBytes,
            completedUploadCount:
              savingProgress.completedUploadCount,
            expectedUploadCount:
              savingProgress.expectedUploadCount,
            title: savingTitle,
            message: savingMessage,
          }),
        );
      },
    }),
    [],
  );

  const handleSaveAnnouncement = useCallback(
    async ({
      payload,
      targetAvatarIds,
    }: SubmitAnnouncementParams): Promise<void> => {
      if (isBusy) {
        return;
      }

      const snapshot =
        createInitialProgressSnapshot();

      setIsSaving(true);
      setCompletedAction(null);
      setProgress(
        createPreparingAnnouncementCreateProgress({
          title: "保存準備中",
          message: "告知情報の保存準備をしています。",
        }),
      );

      try {
        const announcement =
          await saveAnnouncement({
            sales: vm.sales,
            payload,
            createdBy: resolveCreatedBy(),
            targetAvatarIds,
            progressHandlers:
              createProgressHandlers(
                snapshot,
                "告知を保存中",
                "画像転送が完了しました。告知情報を保存しています。",
              ),
          });

        setCompletedAction({
          type: "save",
        });

        setProgress(
          createCompletedAnnouncementCreateProgress({
            transferredBytes:
              snapshot.transferredBytes,
            totalBytes:
              snapshot.totalBytes,
            completedUploadCount:
              snapshot.completedUploadCount,
            expectedUploadCount:
              snapshot.expectedUploadCount,
            title: "保存が完了しました",
            message: announcement.id
              ? "告知の保存が完了しました。"
              : "告知の保存が完了しました。",
          }),
        );
      } catch (error) {
        const message =
          getErrorMessage(error);

        setCompletedAction(null);

        setProgress(
          createFailedAnnouncementCreateProgress(
            message,
            {
              title: "保存に失敗しました",
              message:
                "告知の保存中にエラーが発生しました。",
            },
          ),
        );

        throw error;
      } finally {
        setIsSaving(false);
      }
    },
    [
      isBusy,
      vm.sales,
      resolveCreatedBy,
      createProgressHandlers,
    ],
  );

  const handleSendAnnouncement = useCallback(
    async ({
      payload,
      targetAvatarIds,
    }: SubmitAnnouncementParams): Promise<string> => {
      if (isBusy) {
        return "";
      }

      const snapshot =
        createInitialProgressSnapshot();

      setIsSending(true);
      setCompletedAction(null);
      setProgress(
        createPreparingAnnouncementCreateProgress({
          title: "送信準備中",
          message: "告知の送信準備をしています。",
        }),
      );

      try {
        const announcement =
          await sendAnnouncement({
            sales: vm.sales,
            payload,
            createdBy: resolveCreatedBy(),
            targetAvatarIds,
            progressHandlers:
              createProgressHandlers(
                snapshot,
                "告知を送信中",
                "画像転送が完了しました。告知情報を保存して送信しています。",
              ),
          });

        const announcementId =
          String(
            announcement.id ?? "",
          ).trim();

        if (!announcementId) {
          throw new Error(
            "announcement_id_missing",
          );
        }

        setCompletedAction({
          type: "send",
          announcementId,
        });

        setProgress(
          createCompletedAnnouncementCreateProgress({
            transferredBytes:
              snapshot.transferredBytes,
            totalBytes:
              snapshot.totalBytes,
            completedUploadCount:
              snapshot.completedUploadCount,
            expectedUploadCount:
              snapshot.expectedUploadCount,
            title: "送信が完了しました",
            message: "告知の送信が完了しました。",
          }),
        );

        return announcementId;
      } catch (error) {
        const message =
          getErrorMessage(error);

        setCompletedAction(null);

        setProgress(
          createFailedAnnouncementCreateProgress(
            message,
            {
              title: "送信に失敗しました",
              message:
                "告知の送信中にエラーが発生しました。",
            },
          ),
        );

        throw error;
      } finally {
        setIsSending(false);
      }
    },
    [
      isBusy,
      vm.sales,
      resolveCreatedBy,
      createProgressHandlers,
    ],
  );

  const handleCloseProgress = useCallback(() => {
    if (
      isBusy ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    if (
      progress.phase === "completed" &&
      completedAction
    ) {
      const action =
        completedAction;

      setProgress(
        createInitialAnnouncementCreateProgress(),
      );
      setCompletedAction(null);

      if (action.type === "send") {
        navigate(
          `/sales/announcements/${encodeURIComponent(
            action.announcementId,
          )}`,
          {
            replace: true,
          },
        );

        return;
      }

      navigate("/sales", {
        replace: true,
      });

      return;
    }

    setProgress(
      createInitialAnnouncementCreateProgress(),
    );
    setCompletedAction(null);
  }, [
    isBusy,
    progress.isBlockingNavigation,
    progress.phase,
    completedAction,
    navigate,
  ]);

  const state: UseAnnouncementCreatePageState = {
    isSaving,
    isSending,
    progress,
    progressOpen,
  };

  const handlers: UseAnnouncementCreatePageHandlers = {
    onBack: handleBack,
    onSaveAnnouncement: handleSaveAnnouncement,
    onSendAnnouncement: handleSendAnnouncement,
    onCloseProgress: handleCloseProgress,
  };

  return {
    vm,
    state,
    handlers,
  };
}