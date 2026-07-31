// frontend/console/shell/src/pages/announcementDetailPage.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import InputCard from "../features/announcement/presentation/components/inputCard";
import type { SubmitPayload } from "../features/announcement/presentation/components/inputCard";

import { uploadAnnouncementImages } from "../features/announcement/application/announcement_attachment_service";

import {
  getAnnouncement,
  markAnnouncementPublished,
  updateAnnouncement,
  type Announcement,
  type AnnouncementAttachmentInput,
} from "../features/announcement/infrastructure/announcement_repository_http";

const emptyInputPayload: SubmitPayload = {
  title: "",
  text: "",
  images: [],
  imageUrls: [],
};

function normalizeAvatarIds(
  values: string[] | undefined | null,
): string[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const avatarId = String(
      value ?? "",
    ).trim();

    if (
      !avatarId ||
      seen.has(avatarId)
    ) {
      continue;
    }

    seen.add(avatarId);
    result.push(avatarId);
  }

  return result;
}

function normalizeAttachmentImageUrls(
  values:
    | Announcement["attachmentFiles"]
    | undefined
    | null,
): string[] {
  if (!Array.isArray(values)) {
    return [];
  }

  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const fileUrl = String(
      value?.fileUrl ?? "",
    ).trim();

    const mimeType = String(
      value?.mimeType ?? "",
    )
      .trim()
      .toLowerCase();

    if (!fileUrl) {
      continue;
    }

    if (
      mimeType &&
      !mimeType.startsWith("image/")
    ) {
      continue;
    }

    if (seen.has(fileUrl)) {
      continue;
    }

    seen.add(fileUrl);
    result.push(fileUrl);
  }

  return result;
}

function buildRetainedAttachmentInputs(params: {
  announcement: Announcement;
  imageUrls: string[];
}): AnnouncementAttachmentInput[] {
  const files = Array.isArray(
    params.announcement.attachmentFiles,
  )
    ? params.announcement.attachmentFiles
    : [];

  const retainedUrlSet = new Set(
    params.imageUrls
      .map((url) => String(url ?? "").trim())
      .filter(Boolean),
  );

  const seenObjectPaths = new Set<string>();
  const result: AnnouncementAttachmentInput[] =
    [];

  for (const file of files) {
    const fileUrl = String(
      file?.fileUrl ?? "",
    ).trim();

    if (
      !fileUrl ||
      !retainedUrlSet.has(fileUrl)
    ) {
      continue;
    }

    const fileName = String(
      file?.fileName ?? "",
    ).trim();

    const objectPath = String(
      file?.objectPath ?? "",
    ).trim();

    const mimeType = String(
      file?.mimeType ?? "",
    ).trim();

    const fileSize = Number.isFinite(
      file?.fileSize,
    )
      ? file.fileSize
      : 0;

    if (
      !fileName ||
      !objectPath ||
      seenObjectPaths.has(objectPath)
    ) {
      continue;
    }

    seenObjectPaths.add(objectPath);

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

function mergeAttachmentInputs(
  retainedAttachments: AnnouncementAttachmentInput[],
  uploadedAttachments: AnnouncementAttachmentInput[],
): AnnouncementAttachmentInput[] {
  const seenObjectPaths = new Set<string>();
  const result: AnnouncementAttachmentInput[] =
    [];

  for (const attachment of [
    ...retainedAttachments,
    ...uploadedAttachments,
  ]) {
    const objectPath = String(
      attachment.objectPath ?? "",
    ).trim();

    if (
      !objectPath ||
      seenObjectPaths.has(objectPath)
    ) {
      continue;
    }

    seenObjectPaths.add(objectPath);
    result.push(attachment);
  }

  return result;
}

function getAnnouncementCreatedByName(
  announcement: Announcement | null,
): string {
  return String(
    announcement?.createdByName ||
      announcement?.createdBy ||
      "",
  ).trim();
}

function getAnnouncementUpdatedByName(
  announcement: Announcement | null,
): string {
  return String(
    announcement?.updatedByName ||
      announcement?.updatedBy ||
      "",
  ).trim();
}

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();

  const { announcementId } = useParams<{
    announcementId: string;
  }>();

  const [announcement, setAnnouncement] =
    useState<Announcement | null>(
      null,
    );

  const [inputPayload, setInputPayload] =
    useState<SubmitPayload>(
      emptyInputPayload,
    );

  const [isEditMode, setIsEditMode] =
    useState(false);

  const [isLoading, setIsLoading] =
    useState(false);

  const [isSavingInput, setIsSavingInput] =
    useState(false);

  const [isSendingInput, setIsSendingInput] =
    useState(false);

  const [errorMessage, setErrorMessage] =
    useState<string | null>(null);

  const normalizedAnnouncementId =
    useMemo(() => {
      return String(
        announcementId ?? "",
      ).trim();
    }, [announcementId]);

  const resetFormFromAnnouncement =
    useCallback(
      (source: Announcement) => {
        setInputPayload({
          title: source.title,
          text: source.content,
          images: [],
          imageUrls:
            normalizeAttachmentImageUrls(
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
        "告知IDを取得できませんでした。",
      );
      return;
    }

    setIsLoading(true);
    setErrorMessage(null);

    try {
      const result =
        await getAnnouncement(
          normalizedAnnouncementId,
        );

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

  const reloadAnnouncement = useCallback(
    async (id: string) => {
      const normalizedId = String(
        id ?? "",
      ).trim();

      if (!normalizedId) {
        return null;
      }

      const refreshed =
        await getAnnouncement(
          normalizedId,
        );

      setAnnouncement(refreshed);

      return refreshed;
    },
    [],
  );

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!announcement) {
      setInputPayload(
        emptyInputPayload,
      );
      setIsEditMode(false);
      return;
    }

    resetFormFromAnnouncement(
      announcement,
    );

    if (announcement.published) {
      setIsEditMode(false);
    }
  }, [
    announcement,
    resetFormFromAnnouncement,
  ]);

  const targetAvatarIds = useMemo(() => {
    if (!announcement) {
      return [];
    }

    return normalizeAvatarIds(
      announcement.targetAvatars,
    );
  }, [announcement]);

  const targetAvatarCount =
    targetAvatarIds.length;

  const initialImageUrls = useMemo(() => {
    return normalizeAttachmentImageUrls(
      announcement?.attachmentFiles,
    );
  }, [announcement]);

  const pageTitle =
    announcement?.title ||
    "告知詳細";

  const handleBack = useCallback(() => {
    navigate("/sales");
  }, [navigate]);

  const handleEdit = useCallback(() => {
    if (
      !announcement ||
      announcement.published
    ) {
      return;
    }

    resetFormFromAnnouncement(
      announcement,
    );

    setIsEditMode(true);
  }, [
    announcement,
    resetFormFromAnnouncement,
  ]);

  const handleCancelEdit =
    useCallback(() => {
      if (announcement) {
        resetFormFromAnnouncement(
          announcement,
        );
      }

      setIsEditMode(false);
    }, [
      announcement,
      resetFormFromAnnouncement,
    ]);

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
        imageUrls:
          inputPayload.imageUrls,
      };
    }, [inputPayload]);

  const getUpdatedBy = useCallback(() => {
    return String(
      announcement?.updatedBy ??
        announcement?.createdBy ??
        "",
    ).trim();
  }, [announcement]);

  const updateDraftAnnouncement =
    useCallback(
      async (
        payload: SubmitPayload,
      ): Promise<Announcement | null> => {
        if (
          !announcement ||
          announcement.published
        ) {
          return null;
        }

        const retainedAttachments =
          buildRetainedAttachmentInputs({
            announcement,
            imageUrls:
              payload.imageUrls,
          });

        const uploadedAttachments =
          await uploadAnnouncementImages({
            announcementId:
              announcement.id,
            images: payload.images,
          });

        const attachments =
          mergeAttachmentInputs(
            retainedAttachments,
            uploadedAttachments,
          );

        return updateAnnouncement(
          announcement.id,
          {
            title: payload.title,
            content: payload.text,
            targetToken:
              announcement.targetToken,
            targetAvatars:
              targetAvatarIds,
            attachments,
            updatedBy: getUpdatedBy(),
          },
        );
      },
      [
        announcement,
        getUpdatedBy,
        targetAvatarIds,
      ],
    );

  const handleSave =
    useCallback(async () => {
      if (
        !announcement ||
        announcement.published ||
        isSavingInput ||
        isSendingInput
      ) {
        return;
      }

      const payload =
        buildSubmitPayload();

      setIsSavingInput(true);

      try {
        await updateDraftAnnouncement(
          payload,
        );

        await reloadAnnouncement(
          announcement.id,
        );

        setIsEditMode(false);

        window.alert(
          "告知を保存しました。",
        );
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
      isSavingInput,
      isSendingInput,
      reloadAnnouncement,
      updateDraftAnnouncement,
    ]);

  const handleSend =
    useCallback(async () => {
      if (
        !announcement ||
        announcement.published ||
        isSavingInput ||
        isSendingInput
      ) {
        return;
      }

      const payload =
        buildSubmitPayload();

      setIsSendingInput(true);

      try {
        if (isEditMode) {
          await updateDraftAnnouncement(
            payload,
          );
        }

        await markAnnouncementPublished(
          announcement.id,
          {
            updatedBy: getUpdatedBy(),
          },
        );

        await reloadAnnouncement(
          announcement.id,
        );

        setIsEditMode(false);

        window.alert(
          "告知を送信しました。",
        );
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
      isEditMode,
      isSavingInput,
      isSendingInput,
      reloadAnnouncement,
      updateDraftAnnouncement,
    ]);

  const createdByName =
    getAnnouncementCreatedByName(
      announcement,
    );

  const updatedByName =
    getAnnouncementUpdatedByName(
      announcement,
    );

  const createdAt =
    announcement?.createdAt ?? "";

  const updatedAt =
    announcement?.updatedAt ?? "";

  const canEditOrSend = Boolean(
    announcement &&
      !announcement.published,
  );

  if (
    isLoading &&
    !announcement
  ) {
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
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={handleBack}
      onEdit={
        canEditOrSend &&
        !isEditMode
          ? handleEdit
          : undefined
      }
      onCancel={
        canEditOrSend &&
        isEditMode
          ? handleCancelEdit
          : undefined
      }
      onSave={
        canEditOrSend &&
        isEditMode
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
          mode={
            isEditMode
              ? "edit"
              : "view"
          }
          initialTitle={
            announcement.title
          }
          initialText={
            announcement.content
          }
          initialImages={
            initialImageUrls
          }
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
          targetAvatarCount={
            targetAvatarCount
          }
          createdByName={
            createdByName
          }
          createdAt={createdAt}
          updatedByName={
            updatedByName
          }
          updatedAt={updatedAt}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}