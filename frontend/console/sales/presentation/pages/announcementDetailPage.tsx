// frontend\console\sales\presentation\pages\announcementDetailPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import LogCard from "../../../log/presentation/LogCard";
import InputCard from "../components/inputCard";
import type { SubmitPayload } from "../components/inputCard";
import {
  getAnnouncement,
  markAnnouncementPublished,
  updateAnnouncement,
  type Announcement,
  type AnnouncementAttachmentInput,
} from "../../infrastructure/announcement_repository_http";

const emptyInputPayload: SubmitPayload = {
  title: "",
  text: "",
  images: [],
  imageUrls: [],
};

type AnnouncementAttachmentFileLike = {
  announcementId?: string | null;
  id?: string | null;
  fileName?: string | null;
  fileUrl?: string | null;
  fileSize?: number | null;
  mimeType?: string | null;
  objectPath?: string | null;
};

type AnnouncementWithResolvedFields = Announcement & {
  attachmentFiles?: AnnouncementAttachmentFileLike[];
  createdByName?: string | null;
  updatedByName?: string | null;
};

function normalizeAvatarIds(
  values: string[] | undefined | null,
): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const avatarId = String(value ?? "").trim();
    if (!avatarId) continue;
    if (seen.has(avatarId)) continue;

    seen.add(avatarId);
    result.push(avatarId);
  }

  return result;
}

function normalizeAttachmentImageUrls(
  values: AnnouncementAttachmentFileLike[] | undefined | null,
): string[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const fileUrl = String(value?.fileUrl ?? "").trim();
    const mimeType = String(value?.mimeType ?? "")
      .trim()
      .toLowerCase();

    if (!fileUrl) continue;
    if (mimeType && !mimeType.startsWith("image/")) continue;
    if (seen.has(fileUrl)) continue;

    seen.add(fileUrl);
    result.push(fileUrl);
  }

  return result;
}

function toSafeNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  const numberValue = Number(value);
  if (!Number.isFinite(numberValue)) {
    return 0;
  }

  return numberValue;
}

function getAnnouncementCreatedByName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  return String(
    announcement?.createdByName ||
      announcement?.createdBy ||
      "",
  ).trim();
}

function getAnnouncementUpdatedByName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  return String(
    announcement?.updatedByName ||
      announcement?.updatedBy ||
      "",
  ).trim();
}

function buildRetainedAttachmentInputs(params: {
  announcement: AnnouncementWithResolvedFields;
  imageUrls: string[];
}): AnnouncementAttachmentInput[] {
  const files = Array.isArray(params.announcement.attachmentFiles)
    ? params.announcement.attachmentFiles
    : [];

  const retainedUrlSet = new Set(
    params.imageUrls
      .map((url) => String(url ?? "").trim())
      .filter(Boolean),
  );

  const seen = new Set<string>();
  const result: AnnouncementAttachmentInput[] = [];

  for (const file of files) {
    const fileUrl = String(file?.fileUrl ?? "").trim();
    if (!fileUrl) continue;
    if (!retainedUrlSet.has(fileUrl)) continue;

    const fileName = String(file?.fileName ?? "").trim();
    const objectPath = String(file?.objectPath ?? "").trim();
    const mimeType = String(file?.mimeType ?? "").trim();
    const fileSize = toSafeNumber(file?.fileSize);

    if (!fileName || !objectPath) continue;

    const dedupeKey = objectPath || fileUrl || fileName;
    if (seen.has(dedupeKey)) continue;

    seen.add(dedupeKey);
    result.push({
      fileName,
      fileUrl,
      fileSize,
      mimeType,
      objectPath,
    });
  }

  return result;
}

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();
  const { announcementId } =
    useParams<{ announcementId: string }>();

  const [announcement, setAnnouncement] =
    useState<AnnouncementWithResolvedFields | null>(null);
  const [inputPayload, setInputPayload] =
    useState<SubmitPayload>(emptyInputPayload);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSavingInput, setIsSavingInput] = useState(false);
  const [isSendingInput, setIsSendingInput] = useState(false);
  const [errorMessage, setErrorMessage] =
    useState<string | null>(null);

  const normalizedAnnouncementId = useMemo(() => {
    return String(announcementId ?? "").trim();
  }, [announcementId]);

  const resetFormFromAnnouncement = useCallback(
    (source: AnnouncementWithResolvedFields) => {
      setInputPayload({
        title: source.title,
        text: source.content,
        images: [],
        imageUrls: normalizeAttachmentImageUrls(
          source.attachmentFiles,
        ),
      });
    },
    [],
  );

  const load = useCallback(async () => {
    if (!normalizedAnnouncementId) {
      setAnnouncement(null);
      setErrorMessage(
        "お知らせIDを取得できませんでした。",
      );
      return;
    }

    setIsLoading(true);
    setErrorMessage(null);

    try {
      const result = await getAnnouncement(
        normalizedAnnouncementId,
      );
      setAnnouncement(
        result as AnnouncementWithResolvedFields,
      );
    } catch (error) {
      setAnnouncement(null);
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "お知らせ詳細の取得に失敗しました。",
      );
    } finally {
      setIsLoading(false);
    }
  }, [normalizedAnnouncementId]);

  const reloadAnnouncement = useCallback(
    async (id: string) => {
      const normalizedId = String(id ?? "").trim();
      if (!normalizedId) {
        return null;
      }

      const refreshed = await getAnnouncement(normalizedId);
      const next =
        refreshed as AnnouncementWithResolvedFields;

      setAnnouncement(next);

      return next;
    },
    [],
  );

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

  const targetAvatarIds = useMemo(() => {
    if (!announcement) return [];

    return normalizeAvatarIds(
      announcement.targetAvatars,
    );
  }, [announcement]);

  const initialImageUrls = useMemo(() => {
    return normalizeAttachmentImageUrls(
      announcement?.attachmentFiles,
    );
  }, [announcement]);

  const pageTitle =
    announcement?.title || "お知らせ詳細";

  const handleBack = useCallback(() => {
    navigate("/sales");
  }, [navigate]);

  const handleEdit = useCallback(() => {
    if (!announcement || announcement.published) return;

    resetFormFromAnnouncement(announcement);
    setIsEditMode(true);
  }, [announcement, resetFormFromAnnouncement]);

  const handleCancelEdit = useCallback(() => {
    if (announcement) {
      resetFormFromAnnouncement(announcement);
    }

    setIsEditMode(false);
  }, [announcement, resetFormFromAnnouncement]);

  const handleInputChange = useCallback(
    (payload: SubmitPayload) => {
      setInputPayload(payload);
    },
    [],
  );

  const buildSubmitPayload =
    useCallback((): SubmitPayload => {
      return {
        title: inputPayload.title.trim(),
        text: inputPayload.text.trim(),
        images: inputPayload.images,
        imageUrls: inputPayload.imageUrls,
      };
    }, [inputPayload]);

  const getUpdatedBy = useCallback(() => {
    return String(
      announcement?.updatedBy ??
        announcement?.createdBy ??
        "",
    ).trim();
  }, [announcement]);

  const handleSave = useCallback(async () => {
    if (!announcement) return;
    if (announcement.published) return;
    if (isSavingInput || isSendingInput) return;

    const payload = buildSubmitPayload();

    setIsSavingInput(true);

    try {
      await updateAnnouncement(announcement.id, {
        title: payload.title,
        content: payload.text,
        targetToken: announcement.targetToken,
        targetAvatars: targetAvatarIds,
        published: announcement.published,
        publishedAt: announcement.publishedAt,
        attachments: buildRetainedAttachmentInputs({
          announcement,
          imageUrls: payload.imageUrls,
        }),
        updatedBy: getUpdatedBy(),
      });

      await reloadAnnouncement(announcement.id);

      setIsEditMode(false);
      window.alert("お知らせを保存しました。");
    } catch (error) {
      console.error(
        "[AnnouncementDetailPage] save announcement failed",
        error,
      );
      window.alert(
        error instanceof Error
          ? error.message
          : "お知らせの保存に失敗しました。",
      );
    } finally {
      setIsSavingInput(false);
    }
  }, [
    announcement,
    buildSubmitPayload,
    getUpdatedBy,
    isSavingInput,
    isSendingInput,
    reloadAnnouncement,
    targetAvatarIds,
  ]);

  const handleSend = useCallback(async () => {
    if (!announcement) return;
    if (announcement.published) return;
    if (isSavingInput || isSendingInput) return;

    const payload = buildSubmitPayload();

    setIsSendingInput(true);

    try {
      if (isEditMode) {
        await updateAnnouncement(announcement.id, {
          title: payload.title,
          content: payload.text,
          targetToken: announcement.targetToken,
          targetAvatars: targetAvatarIds,
          published: announcement.published,
          publishedAt: announcement.publishedAt,
          attachments: buildRetainedAttachmentInputs({
            announcement,
            imageUrls: payload.imageUrls,
          }),
          updatedBy: getUpdatedBy(),
        });
      }

      await markAnnouncementPublished(announcement.id, {
        updatedBy: getUpdatedBy(),
      });

      await reloadAnnouncement(announcement.id);

      setIsEditMode(false);
      window.alert("お知らせを送信しました。");
    } catch (error) {
      console.error(
        "[AnnouncementDetailPage] send announcement failed",
        error,
      );
      window.alert(
        error instanceof Error
          ? error.message
          : "お知らせの送信に失敗しました。",
      );
    } finally {
      setIsSendingInput(false);
    }
  }, [
    announcement,
    buildSubmitPayload,
    getUpdatedBy,
    isEditMode,
    isSavingInput,
    isSendingInput,
    reloadAnnouncement,
    targetAvatarIds,
  ]);

  const createdByName =
    getAnnouncementCreatedByName(announcement);
  const updatedByName =
    getAnnouncementUpdatedByName(announcement);
  const createdAt = announcement?.createdAt ?? "";
  const updatedAt = announcement?.updatedAt ?? "";

  const canEditOrSend = Boolean(
    announcement && !announcement.published,
  );

  if (isLoading && !announcement) {
    return (
      <PageStyle
        layout="single"
        title="お知らせ詳細"
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
        title="お知らせ詳細"
        onBack={handleBack}
      >
        <p className="p-4 text-sm text-red-600">
          {errorMessage}
        </p>
      </PageStyle>
    );
  }

  if (!announcement) {
    return (
      <PageStyle
        layout="single"
        title="お知らせ詳細"
        onBack={handleBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なお知らせ詳細がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={handleBack}
      onEdit={
        canEditOrSend && !isEditMode
          ? handleEdit
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
          onChange={
            isEditMode ? handleInputChange : undefined
          }
        />
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          mode="view"
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