// frontend/console/shell/src/pages/announcementDetailPage.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import InputCard from "../features/announcement/presentation/components/inputCard";
import type { SubmitPayload } from "../features/announcement/presentation/components/inputCard";

import { uploadAnnouncementImages } from "../features/announcement/application/announcement_attachment_service";

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

const emptyInputPayload: SubmitPayload = {
  title: "",
  text: "",
  images: [],
  imageUrls: [],
};

function getAttachmentImageUrls(
  files: AnnouncementAttachmentFile[] | undefined,
): string[] {
  if (!files) {
    return [];
  }

  return files
    .filter((file) => !file.mimeType || file.mimeType.startsWith("image/"))
    .map((file) => file.fileUrl);
}

function buildRetainedAttachmentInputs(params: {
  announcement: AnnouncementDetail;
  imageUrls: string[];
}): AnnouncementAttachmentInput[] {
  const retainedUrlSet = new Set(params.imageUrls);

  return (params.announcement.attachmentFiles ?? [])
    .filter((file) => retainedUrlSet.has(file.fileUrl))
    .map((file) => ({
      fileName: file.fileName,
      fileUrl: file.fileUrl,
      fileSize: file.fileSize,
      mimeType: file.mimeType,
      objectPath: file.objectPath,
    }));
}

function mergeAttachmentInputs(
  retainedAttachments: AnnouncementAttachmentInput[],
  uploadedAttachments: AnnouncementAttachmentInput[],
): AnnouncementAttachmentInput[] {
  const attachments = new Map<string, AnnouncementAttachmentInput>();

  for (const attachment of [...retainedAttachments, ...uploadedAttachments]) {
    attachments.set(attachment.objectPath, attachment);
  }

  return [...attachments.values()];
}

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();
  const { announcementId } = useParams<{ announcementId: string }>();

  const [announcement, setAnnouncement] = useState<AnnouncementDetail | null>(null);
  const [inputPayload, setInputPayload] = useState<SubmitPayload>(emptyInputPayload);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSavingInput, setIsSavingInput] = useState(false);
  const [isSendingInput, setIsSendingInput] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const normalizedAnnouncementId = useMemo(
    () => String(announcementId ?? "").trim(),
    [announcementId],
  );

  const resetFormFromAnnouncement = useCallback((source: AnnouncementDetail) => {
    setInputPayload({
      title: source.title,
      text: source.content,
      images: [],
      imageUrls: getAttachmentImageUrls(source.attachmentFiles),
    });
  }, []);

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

  const targetAvatarIds = useMemo(
    () => announcement?.targetAvatars ?? [],
    [announcement],
  );

  const targetAvatarCount = targetAvatarIds.length;

  const initialImageUrls = useMemo(
    () => getAttachmentImageUrls(announcement?.attachmentFiles),
    [announcement],
  );

  const pageTitle = announcement?.title || "告知詳細";

  const handleBack = useCallback(() => {
    navigate("/sales");
  }, [navigate]);

  const handleEdit = useCallback(() => {
    if (!announcement || announcement.published || isDeleting) {
      return;
    }

    resetFormFromAnnouncement(announcement);
    setIsEditMode(true);
  }, [announcement, isDeleting, resetFormFromAnnouncement]);

  const handleCancelEdit = useCallback(() => {
    if (isDeleting) {
      return;
    }

    if (announcement) {
      resetFormFromAnnouncement(announcement);
    }

    setIsEditMode(false);
  }, [announcement, isDeleting, resetFormFromAnnouncement]);

  const handleDelete = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      !isEditMode ||
      isSavingInput ||
      isSendingInput ||
      isDeleting
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
      console.error(
        "[AnnouncementDetailPage] delete announcement failed",
        error,
      );

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
  ]);

  const handleInputChange = useCallback((payload: SubmitPayload) => {
    setInputPayload(payload);
  }, []);

  const buildSubmitPayload = useCallback((): SubmitPayload => {
    return {
      title: inputPayload.title.trim(),
      text: inputPayload.text.trim(),
      images: inputPayload.images,
      imageUrls: inputPayload.imageUrls,
    };
  }, [inputPayload]);

  const getUpdatedBy = useCallback(() => {
    return announcement?.updatedBy ?? announcement?.createdBy ?? "";
  }, [announcement]);

  const updateDraftAnnouncement = useCallback(
    async (payload: SubmitPayload): Promise<void> => {
      if (!announcement || announcement.published) {
        return;
      }

      const retainedAttachments = buildRetainedAttachmentInputs({
        announcement,
        imageUrls: payload.imageUrls,
      });

      const uploadedAttachments = await uploadAnnouncementImages({
        announcementId: announcement.id,
        images: payload.images,
      });

      const attachments = mergeAttachmentInputs(
        retainedAttachments,
        uploadedAttachments,
      );

      await updateAnnouncement(announcement.id, {
        title: payload.title,
        content: payload.text,
        targetToken: announcement.targetToken,
        targetAvatars: targetAvatarIds,
        attachments,
        updatedBy: getUpdatedBy(),
      });
    },
    [announcement, getUpdatedBy, targetAvatarIds],
  );

  const handleSave = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      isSavingInput ||
      isSendingInput ||
      isDeleting
    ) {
      return;
    }

    const payload = buildSubmitPayload();
    setIsSavingInput(true);

    try {
      await updateDraftAnnouncement(payload);
      await reloadAnnouncement(announcement.id);

      setIsEditMode(false);
      window.alert("告知を保存しました。");
    } catch (error) {
      console.error(
        "[AnnouncementDetailPage] save announcement failed",
        error,
      );

      window.alert(
        error instanceof Error
          ? error.message
          : "告知の保存に失敗しました。",
      );
    } finally {
      setIsSavingInput(false);
    }
  }, [
    announcement,
    buildSubmitPayload,
    isDeleting,
    isSavingInput,
    isSendingInput,
    reloadAnnouncement,
    updateDraftAnnouncement,
  ]);

  const handleSend = useCallback(async () => {
    if (
      !announcement ||
      announcement.published ||
      isSavingInput ||
      isSendingInput ||
      isDeleting
    ) {
      return;
    }

    const payload = buildSubmitPayload();
    setIsSendingInput(true);

    try {
      if (isEditMode) {
        await updateDraftAnnouncement(payload);
      }

      await markAnnouncementPublished(announcement.id, {
        updatedBy: getUpdatedBy(),
      });

      await reloadAnnouncement(announcement.id);

      setIsEditMode(false);
      window.alert("告知を送信しました。");
    } catch (error) {
      console.error(
        "[AnnouncementDetailPage] send announcement failed",
        error,
      );

      window.alert(
        error instanceof Error
          ? error.message
          : "告知の送信に失敗しました。",
      );
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
    reloadAnnouncement,
    updateDraftAnnouncement,
  ]);

  const createdByName = announcement?.createdByName ?? "";
  const updatedByName = announcement?.updatedByName ?? "";
  const createdAt = announcement?.createdAt ?? "";
  const updatedAt = announcement?.updatedAt ?? "";

  const canEditOrSend = Boolean(
    announcement && !announcement.published && !isDeleting,
  );

  const canDelete = Boolean(
    announcement && !announcement.published && isEditMode,
  );

  if (isLoading && !announcement) {
    return (
      <PageStyle layout="single" title="告知詳細" onBack={handleBack}>
        <p className="p-4 text-sm text-muted-foreground">
          読み込み中です。
        </p>
      </PageStyle>
    );
  }

  if (errorMessage) {
    return (
      <PageStyle layout="single" title="告知詳細" onBack={handleBack}>
        <p className="p-4 text-sm text-red-600">
          {errorMessage}
        </p>
      </PageStyle>
    );
  }

  if (!announcement) {
    return (
      <PageStyle layout="single" title="告知詳細" onBack={handleBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能な告知詳細がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={handleBack}
      onEdit={canEditOrSend && !isEditMode ? handleEdit : undefined}
      onDelete={canDelete ? handleDelete : undefined}
      onCancel={canEditOrSend && isEditMode ? handleCancelEdit : undefined}
      onSave={canEditOrSend && isEditMode ? handleSave : undefined}
      isSaving={isSavingInput}
      onSend={canEditOrSend ? handleSend : undefined}
      isSending={isSendingInput}
    >
      <div className="space-y-4">
        <InputCard
          title="入力"
          mode={isEditMode ? "edit" : "view"}
          initialTitle={announcement.title}
          initialText={announcement.content}
          initialImages={initialImageUrls}
          saving={isSavingInput}
          sending={isSendingInput}
          onChange={isEditMode ? handleInputChange : undefined}
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
  );
}