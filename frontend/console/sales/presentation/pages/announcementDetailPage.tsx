// frontend/console/sales/presentation/pages/announcementDetailPage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import LogCard from "../../../log/presentation/LogCard";
import InputCard from "../components/inputCard";
import type { SubmitPayload } from "../components/inputCard";
import SalesOwnersCard, {
  type SalesOwnerItem,
} from "../components/salesOwnersCard";
import {
  getAnnouncement,
  markAnnouncementPublished,
  updateAnnouncement,
  type Announcement,
} from "../../infrastructure/announcement_repository_http";

const emptyInputPayload: SubmitPayload = {
  title: "",
  text: "",
  images: [],
};

function normalizeAvatarIds(values: string[] | undefined | null): string[] {
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

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();
  const { announcementId } = useParams<{ announcementId: string }>();

  const [announcement, setAnnouncement] = useState<Announcement | null>(null);
  const [inputPayload, setInputPayload] =
    useState<SubmitPayload>(emptyInputPayload);
  const [selectedAvatarIds, setSelectedAvatarIds] = useState<string[]>([]);
  const [isEditMode, setIsEditMode] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isSavingInput, setIsSavingInput] = useState(false);
  const [isSendingInput, setIsSendingInput] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const normalizedAnnouncementId = useMemo(() => {
    return String(announcementId ?? "").trim();
  }, [announcementId]);

  const resetFormFromAnnouncement = useCallback((source: Announcement) => {
    setInputPayload({
      title: source.title,
      text: source.content,
      images: [],
    });

    setSelectedAvatarIds(normalizeAvatarIds(source.targetAvatars));
  }, []);

  const load = useCallback(async () => {
    if (!normalizedAnnouncementId) {
      setAnnouncement(null);
      setErrorMessage("告知IDが取得できませんでした。");
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

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!announcement) {
      setInputPayload(emptyInputPayload);
      setSelectedAvatarIds([]);
      setIsEditMode(false);
      return;
    }

    resetFormFromAnnouncement(announcement);

    if (announcement.published) {
      setIsEditMode(false);
    }
  }, [announcement, resetFormFromAnnouncement]);

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

  const handleInputChange = useCallback((payload: SubmitPayload) => {
    setInputPayload(payload);
  }, []);

  const handleSelectionChange = useCallback((avatarIds: string[]) => {
    setSelectedAvatarIds(normalizeAvatarIds(avatarIds));
  }, []);

  const buildSubmitPayload = useCallback((): SubmitPayload => {
    return {
      title: inputPayload.title.trim(),
      text: inputPayload.text.trim(),
      images: inputPayload.images,
    };
  }, [inputPayload]);

  const getUpdatedBy = useCallback(() => {
    return String(
      announcement?.updatedBy ?? announcement?.createdBy ?? "",
    ).trim();
  }, [announcement]);

  const handleSave = useCallback(async () => {
    if (!announcement) return;
    if (announcement.published) return;
    if (isSavingInput || isSendingInput) return;

    const payload = buildSubmitPayload();

    setIsSavingInput(true);

    try {
      const result = await updateAnnouncement(announcement.id, {
        title: payload.title,
        content: payload.text,
        targetToken: announcement.targetToken,
        targetAvatars: selectedAvatarIds,
        attachments: announcement.attachments,
        published: announcement.published,
        publishedAt: announcement.publishedAt,
        updatedBy: getUpdatedBy(),
      });

      setAnnouncement(result);
      setIsEditMode(false);
      window.alert("告知を保存しました。");
    } catch (error) {
      console.error("[AnnouncementDetailPage] save announcement failed", error);
      window.alert(
        error instanceof Error ? error.message : "告知の保存に失敗しました。",
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
    selectedAvatarIds,
  ]);

  const handleSend = useCallback(async () => {
    if (!announcement) return;
    if (announcement.published) return;
    if (isSavingInput || isSendingInput) return;

    const payload = buildSubmitPayload();

    setIsSendingInput(true);

    try {
      let targetAnnouncement = announcement;

      if (isEditMode) {
        targetAnnouncement = await updateAnnouncement(announcement.id, {
          title: payload.title,
          content: payload.text,
          targetToken: announcement.targetToken,
          targetAvatars: selectedAvatarIds,
          attachments: announcement.attachments,
          published: announcement.published,
          publishedAt: announcement.publishedAt,
          updatedBy: getUpdatedBy(),
        });
      }

      const result = await markAnnouncementPublished(targetAnnouncement.id, {
        updatedBy: getUpdatedBy(),
      });

      setAnnouncement(result);
      setIsEditMode(false);
      window.alert("告知を送信しました。");
    } catch (error) {
      console.error("[AnnouncementDetailPage] send announcement failed", error);
      window.alert(
        error instanceof Error ? error.message : "告知の送信に失敗しました。",
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
    selectedAvatarIds,
  ]);

  const targetAvatarIds = useMemo(() => {
    if (isEditMode) {
      return normalizeAvatarIds(selectedAvatarIds);
    }

    if (!announcement) return [];

    return normalizeAvatarIds(announcement.targetAvatars);
  }, [announcement, isEditMode, selectedAvatarIds]);

  const owners = useMemo<SalesOwnerItem[]>(() => {
    return targetAvatarIds.map((avatarId) => ({
      avatarId,
      avatarName: avatarId,
      avatarIconUrl: "",
      productName: "",
      followerCount: 0,
      followingCount: 0,
      postCount: 0,
    }));
  }, [targetAvatarIds]);

  const createdByName = announcement?.createdBy ?? "";
  const updatedByName = announcement?.updatedBy ?? "";
  const createdAt = announcement?.createdAt ?? "";
  const updatedAt = announcement?.updatedAt ?? "";

  const canEditOrSend = Boolean(announcement && !announcement.published);

  if (isLoading && !announcement) {
    return (
      <PageStyle layout="single" title="告知詳細" onBack={handleBack}>
        <p className="p-4 text-sm text-muted-foreground">読み込み中です。</p>
      </PageStyle>
    );
  }

  if (errorMessage) {
    return (
      <PageStyle layout="single" title="告知詳細" onBack={handleBack}>
        <p className="p-4 text-sm text-red-600">{errorMessage}</p>
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
      title="告知詳細"
      onBack={handleBack}
      onEdit={canEditOrSend && !isEditMode ? handleEdit : undefined}
      onCancel={canEditOrSend && isEditMode ? handleCancelEdit : undefined}
      onSave={canEditOrSend && isEditMode ? handleSave : undefined}
      isSaving={isSavingInput}
      onSend={canEditOrSend ? handleSend : undefined}
      isSending={isSendingInput}
    >
      <div className="space-y-4">
        <div style={{ marginTop: 16 }}>
          <SalesOwnersCard
            title="送信対象アバター"
            mode={isEditMode ? "edit" : "view"}
            owners={owners}
            selectedAvatarIds={targetAvatarIds}
            onSelectionChange={isEditMode ? handleSelectionChange : undefined}
          />
        </div>

        <InputCard
          title="入力"
          mode={isEditMode ? "edit" : "view"}
          initialTitle={announcement.title}
          initialText={announcement.content}
          initialImages={[]}
          saving={isSavingInput}
          sending={isSendingInput}
          onChange={isEditMode ? handleInputChange : undefined}
        />
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          mode="view"
          assigneeId=""
          assigneeName=""
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