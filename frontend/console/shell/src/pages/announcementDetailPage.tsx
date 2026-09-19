// frontend/console/shell/src/pages/announcementDetailPage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import InputCard from "../features/announcement/presentation/components/inputCard";
import AnnouncementCreateProgressModal from "../features/announcement/presentation/components/announcementCreateProgressModal";
import { ErrorMessage } from "../shared/ui/error";

import type {
  AnnouncementInputAttachment,
  AnnouncementInputPayload,
} from "../features/announcement/application/announcement_input";

import { uploadAnnouncementImages } from "../features/announcement/application/announcement_attachment_service";

import {
  createCompletedAnnouncementCreateProgress,
  createFailedAnnouncementCreateProgress,
  createInitialAnnouncementCreateProgress,
  createPreparingAnnouncementCreateProgress,
  createSavingAnnouncementCreateProgress,
  createUploadingAnnouncementCreateProgress,
  isAnnouncementCreateProgressVisible,
  type AnnouncementCreateProgress,
} from "../features/announcement/presentation/model/announcementCreateProgress";

import {
  deleteAnnouncement,
  getAnnouncement,
  markAnnouncementPublished,
  updateAnnouncement,
} from "../features/announcement/infrastructure/announcement_repository_http";

import type {
  AnnouncementAttachmentFile,
  AnnouncementAttachmentInput,
  AnnouncementDetail,
} from "../shared/types/announcements";

const emptyInputPayload: AnnouncementInputPayload = {
  title: "",
  text: "",
  attachments: [],
};

type DraftUpdateResult = {
  hadImageUpload: boolean;
  transferredBytes: number;
  totalBytes: number;
  completedUploadCount: number;
  expectedUploadCount: number;
};

function toInputAttachments(
  files: AnnouncementAttachmentFile[] | undefined,
): AnnouncementInputAttachment[] {
  return (files ?? [])
    .filter((file) => !file.mimeType || file.mimeType.startsWith("image/"))
    .map((file) => ({
      ...file,
      type: "existing",
    }));
}

function hasNewImages(payload: AnnouncementInputPayload): boolean {
  return payload.attachments.some(
    (attachment) => attachment.type === "new",
  );
}

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();
  const { announcementId } = useParams<{ announcementId: string }>();

  const [announcement, setAnnouncement] =
    useState<AnnouncementDetail | null>(null);
  const [inputPayload, setInputPayload] =
    useState<AnnouncementInputPayload>(emptyInputPayload);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSavingInput, setIsSavingInput] = useState(false);
  const [isSendingInput, setIsSendingInput] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [progress, setProgress] = useState<AnnouncementCreateProgress>(
    createInitialAnnouncementCreateProgress,
  );

  const normalizedAnnouncementId = useMemo(
    () => String(announcementId ?? "").trim(),
    [announcementId],
  );

  const progressOpen = isAnnouncementCreateProgressVisible(progress);

  const resetFormFromAnnouncement = useCallback(
    (source: AnnouncementDetail) => {
      setInputPayload({
        title: source.title,
        text: source.content,
        attachments: toInputAttachments(source.attachmentFiles),
      });
    },
    [],
  );

  const load = useCallback(async () => {
    if (!normalizedAnnouncementId) {
      setAnnouncement(null);
      setErrorMessage("告知IDを取得できませんでした。");
      return;
    }

    setIsLoading(true);
    setErrorMessage(null);

    try {
      const result = await getAnnouncement(normalizedAnnouncementId);
      setAnnouncement(result);
    } catch (error) {
      setAnnouncement(null);
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "告知詳細の取得に失敗しました。",
      );
    } finally {
      setIsLoading(false);
    }
  }, [normalizedAnnouncementId]);

  const reloadAnnouncement = useCallback(async (id: string) => {
    if (!id) {
      return null;
    }

    const refreshed = await getAnnouncement(id);
    setAnnouncement(refreshed);

    return refreshed;
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!announcement) {
      setInputPayload(emptyInputPayload);
      setIsEditMode(false);
      return;
    }

    resetFormFromAnnouncement(announcement);

    if (announcement.published) {
      setIsEditMode(false);
    }
  }, [announcement, resetFormFromAnnouncement]);

  useEffect(() => {
    if (!progress.isBlockingNavigation) {
      return;
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [progress.isBlockingNavigation]);

  const targetAvatarIds = useMemo(
    () => announcement?.targetAvatars ?? [],
    [announcement],
  );

  const targetAvatarCount = targetAvatarIds.length;

  const initialAttachments = useMemo(
    () => toInputAttachments(announcement?.attachmentFiles),
    [announcement],
  );

  const pageTitle = announcement?.title || "告知詳細";

  const handleBack = useCallback(() => {
    if (progress.isBlockingNavigation) {
      return;
    }

    navigate("/sales");
  }, [navigate, progress.isBlockingNavigation]);

  const handleEdit = useCallback(() => {
    if (
      !announcement ||
      announcement.published ||
      isDeleting ||
      isSavingInput ||
      isSendingInput ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    resetFormFromAnnouncement(announcement);
    setIsEditMode(true);
  }, [
    announcement,
    isDeleting,
    isSavingInput,
    isSendingInput,
    progress.isBlockingNavigation,
    resetFormFromAnnouncement,
  ]);

  const handleCancelEdit = useCallback(() => {
    if (
      isDeleting ||
      isSavingInput ||
      isSendingInput ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    if (announcement) {
      resetFormFromAnnouncement(announcement);
    }

    setIsEditMode(false);
  }, [
    announcement,
    isDeleting,
    isSavingInput,
    isSendingInput,
    progress.isBlockingNavigation,
    resetFormFromAnnouncement,
  ]);

  const handleDelete = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      !isEditMode ||
      isSavingInput ||
      isSendingInput ||
      isDeleting ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    const confirmed = window.confirm(
      "この告知を削除しますか？\n関連する画像と告知データも削除されます。",
    );

    if (!confirmed) {
      return;
    }

    setIsDeleting(true);

    try {
      await deleteAnnouncement(announcement.id);
      window.alert("告知を削除しました。");
      navigate("/sales");
    } catch (error) {
      window.alert(
        error instanceof Error
          ? error.message
          : "告知の削除に失敗しました。",
      );
    } finally {
      setIsDeleting(false);
    }
  }, [
    announcement,
    isDeleting,
    isEditMode,
    isSavingInput,
    isSendingInput,
    navigate,
    progress.isBlockingNavigation,
  ]);

  const handleInputChange = useCallback(
    (payload: AnnouncementInputPayload) => {
      setInputPayload(payload);
    },
    [],
  );

  const buildSubmitPayload = useCallback(
    (): AnnouncementInputPayload => ({
      title: inputPayload.title.trim(),
      text: inputPayload.text.trim(),
      attachments: inputPayload.attachments,
    }),
    [inputPayload],
  );

  const getUpdatedBy = useCallback(() => {
    return announcement?.updatedBy ?? announcement?.createdBy ?? "";
  }, [announcement]);

  const updateDraftAnnouncement = useCallback(
    async (payload: AnnouncementInputPayload): Promise<DraftUpdateResult> => {
      if (!announcement || announcement.published) {
        return {
          hadImageUpload: false,
          transferredBytes: 0,
          totalBytes: 0,
          completedUploadCount: 0,
          expectedUploadCount: 0,
        };
      }

      const existingAttachments: AnnouncementAttachmentInput[] = [];
      const newFiles: File[] = [];

      for (const attachment of payload.attachments) {
        if (attachment.type === "new") {
          newFiles.push(attachment.file);
          continue;
        }

        existingAttachments.push({
          fileName: attachment.fileName,
          fileUrl: attachment.fileUrl,
          fileSize: attachment.fileSize,
          mimeType: attachment.mimeType,
          objectPath: attachment.objectPath,
        });
      }

      const hadImageUpload = newFiles.length > 0;
      let transferredBytes = 0;
      let totalBytes = 0;
      let completedUploadCount = 0;
      let expectedUploadCount = 0;
      let uploadedAttachments: AnnouncementAttachmentInput[] = [];

      if (hadImageUpload) {
        setProgress(
          createPreparingAnnouncementCreateProgress({
            title: "保存準備中",
            message: "告知画像の転送準備をしています。",
          }),
        );

        uploadedAttachments = await uploadAnnouncementImages({
          announcementId: announcement.id,
          images: newFiles,
          onProgress: (uploadProgress) => {
            transferredBytes = uploadProgress.transferredBytes;
            totalBytes = uploadProgress.totalBytes;
            completedUploadCount = uploadProgress.completedUploadCount;
            expectedUploadCount = uploadProgress.expectedUploadCount;

            setProgress(
              createUploadingAnnouncementCreateProgress({
                fileName: uploadProgress.fileName,
                transferredBytes: uploadProgress.transferredBytes,
                totalBytes: uploadProgress.totalBytes,
                completedUploadCount: uploadProgress.completedUploadCount,
                expectedUploadCount: uploadProgress.expectedUploadCount,
                title: "画像を転送中",
                message:
                  "画像転送が完了するまで、この画面を閉じたり別のページへ移動したりしないでください。",
              }),
            );
          },
        });

        setProgress(
          createSavingAnnouncementCreateProgress({
            transferredBytes,
            totalBytes,
            completedUploadCount,
            expectedUploadCount,
            title: "告知を保存中",
            message: "画像転送が完了しました。告知情報を保存しています。",
          }),
        );
      }

      const attachments = [
        ...existingAttachments,
        ...uploadedAttachments,
      ];

      await updateAnnouncement(announcement.id, {
        title: payload.title,
        content: payload.text,
        targetToken: announcement.targetToken,
        targetAvatars: targetAvatarIds,
        attachments,
        updatedBy: getUpdatedBy(),
      });

      return {
        hadImageUpload,
        transferredBytes,
        totalBytes,
        completedUploadCount,
        expectedUploadCount,
      };
    },
    [announcement, getUpdatedBy, targetAvatarIds],
  );

  const handleSave = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      isSavingInput ||
      isSendingInput ||
      isDeleting ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    const payload = buildSubmitPayload();
    const includesNewImages = hasNewImages(payload);

    setIsSavingInput(true);

    try {
      const result = await updateDraftAnnouncement(payload);
      await reloadAnnouncement(announcement.id);

      setIsEditMode(false);

      if (result.hadImageUpload) {
        setProgress(
          createCompletedAnnouncementCreateProgress({
            transferredBytes: result.transferredBytes,
            totalBytes: result.totalBytes,
            completedUploadCount: result.completedUploadCount,
            expectedUploadCount: result.expectedUploadCount,
            title: "保存が完了しました",
            message: "告知の保存が完了しました。",
          }),
        );
      } else {
        window.alert("告知を保存しました。");
      }
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "告知の保存に失敗しました。";

      if (includesNewImages) {
        setProgress(
          createFailedAnnouncementCreateProgress(
            message,
            {
              title: "保存に失敗しました",
              message: "告知の保存中にエラーが発生しました。",
            },
          ),
        );
      } else {
        window.alert(message);
      }
    } finally {
      setIsSavingInput(false);
    }
  }, [
    announcement,
    buildSubmitPayload,
    isDeleting,
    isSavingInput,
    isSendingInput,
    progress.isBlockingNavigation,
    reloadAnnouncement,
    updateDraftAnnouncement,
  ]);

  const handleSend = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      isSavingInput ||
      isSendingInput ||
      isDeleting ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    const payload = buildSubmitPayload();
    const includesNewImages = isEditMode && hasNewImages(payload);

    setIsSendingInput(true);

    try {
      let updateResult: DraftUpdateResult | null = null;

      if (isEditMode) {
        updateResult = await updateDraftAnnouncement(payload);
      }

      if (updateResult?.hadImageUpload) {
        setProgress(
          createSavingAnnouncementCreateProgress({
            transferredBytes: updateResult.transferredBytes,
            totalBytes: updateResult.totalBytes,
            completedUploadCount: updateResult.completedUploadCount,
            expectedUploadCount: updateResult.expectedUploadCount,
            title: "告知を送信中",
            message:
              "画像転送が完了しました。告知の送信処理を続けています。",
          }),
        );
      }

      await markAnnouncementPublished(announcement.id, {
        updatedBy: getUpdatedBy(),
      });

      await reloadAnnouncement(announcement.id);
      setIsEditMode(false);

      if (updateResult?.hadImageUpload) {
        setProgress(
          createCompletedAnnouncementCreateProgress({
            transferredBytes: updateResult.transferredBytes,
            totalBytes: updateResult.totalBytes,
            completedUploadCount: updateResult.completedUploadCount,
            expectedUploadCount: updateResult.expectedUploadCount,
            title: "送信が完了しました",
            message: "告知の送信が完了しました。",
          }),
        );
      } else {
        window.alert("告知を送信しました。");
      }
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "告知の送信に失敗しました。";

      if (includesNewImages) {
        setProgress(
          createFailedAnnouncementCreateProgress(
            message,
            {
              title: "送信に失敗しました",
              message: "告知の送信中にエラーが発生しました。",
            },
          ),
        );
      } else {
        window.alert(message);
      }
    } finally {
      setIsSendingInput(false);
    }
  }, [
    announcement,
    buildSubmitPayload,
    getUpdatedBy,
    isDeleting,
    isEditMode,
    isSavingInput,
    isSendingInput,
    progress.isBlockingNavigation,
    reloadAnnouncement,
    updateDraftAnnouncement,
  ]);

  const handleCloseProgress = useCallback(() => {
    if (
      isSavingInput ||
      isSendingInput ||
      progress.isBlockingNavigation
    ) {
      return;
    }

    setProgress(createInitialAnnouncementCreateProgress());
  }, [
    isSavingInput,
    isSendingInput,
    progress.isBlockingNavigation,
  ]);

  const createdByName = announcement?.createdByName ?? "";
  const updatedByName = announcement?.updatedByName ?? "";
  const createdAt = announcement?.createdAt ?? "";
  const updatedAt = announcement?.updatedAt ?? "";

  const canEditOrSend = Boolean(
    announcement &&
      !announcement.published &&
      !isDeleting &&
      !progress.isBlockingNavigation,
  );

  const canDelete = Boolean(
    announcement &&
      !announcement.published &&
      isEditMode &&
      !isSavingInput &&
      !isSendingInput &&
      !progress.isBlockingNavigation,
  );

  if (isLoading && !announcement) {
    return (
      <PageStyle
        layout="single"
        title="告知詳細"
        onBack={handleBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          読み込み中です。
        </p>
      </PageStyle>
    );
  }

  if (errorMessage) {
    return (
      <PageStyle
        layout="single"
        title="告知詳細"
        onBack={handleBack}
      >
        <ErrorMessage className="p-4">
          {errorMessage}
        </ErrorMessage>
      </PageStyle>
    );
  }

  if (!announcement) {
    return (
      <PageStyle
        layout="single"
        title="告知詳細"
        onBack={handleBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          表示可能な告知詳細がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title={pageTitle}
        onBack={handleBack}
        onEdit={
          canEditOrSend && !isEditMode
            ? handleEdit
            : undefined
        }
        onDelete={
          canDelete
            ? handleDelete
            : undefined
        }
        onCancel={
          canEditOrSend && isEditMode
            ? handleCancelEdit
            : undefined
        }
        onSave={
          canEditOrSend && isEditMode
            ? handleSave
            : undefined
        }
        isSaving={isSavingInput}
        onSend={
          canEditOrSend
            ? handleSend
            : undefined
        }
        isSending={isSendingInput}
      >
        <div className="space-y-4">
          <InputCard
            title="入力"
            mode={isEditMode ? "edit" : "view"}
            initialTitle={announcement.title}
            initialText={announcement.content}
            initialAttachments={initialAttachments}
            saving={isSavingInput}
            sending={isSendingInput}
            onChange={
              isEditMode
                ? handleInputChange
                : undefined
            }
          />
        </div>

        <div className="space-y-4">
          <AdminCard
            title="管理情報"
            mode="view"
            targetAvatarCount={targetAvatarCount}
            createdByName={createdByName}
            createdAt={createdAt}
            updatedByName={updatedByName}
            updatedAt={updatedAt}
          />

          <LogCard title="更新ログ" />
        </div>
      </PageStyle>

      <AnnouncementCreateProgressModal
        open={progressOpen}
        progress={progress}
        onClose={
          isSavingInput ||
          isSendingInput ||
          progress.isBlockingNavigation
            ? undefined
            : handleCloseProgress
        }
      />
    </>
  );
}