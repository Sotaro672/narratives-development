// frontend\console\sales\presentation\pages\announcementDetailPage.tsx
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

type AnnouncementTargetAvatarDetailLike = {
  avatarId?: string | null;
  avatarName?: string | null;
  avatarIcon?: string | null;
  avatarIconUrl?: string | null;
  followerCount?: number | null;
  followingCount?: number | null;
  postCount?: number | null;
};

type AnnouncementProductBlueprintLike = {
  productBlueprintId?: string | null;
  productName?: string | null;
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
  tokenName?: string | null;
  targetAvatarDetails?: AnnouncementTargetAvatarDetailLike[];
  productBlueprints?: AnnouncementProductBlueprintLike[];
  attachmentFiles?: AnnouncementAttachmentFileLike[];
  createdByName?: string | null;
  updatedByName?: string | null;
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

function normalizeAvatarDetails(
  values: AnnouncementTargetAvatarDetailLike[] | undefined | null,
): AnnouncementTargetAvatarDetailLike[] {
  if (!Array.isArray(values)) return [];

  const seen = new Set<string>();
  const result: AnnouncementTargetAvatarDetailLike[] = [];

  for (const value of values) {
    const avatarId = String(value.avatarId ?? "").trim();
    if (!avatarId) continue;
    if (seen.has(avatarId)) continue;

    seen.add(avatarId);
    result.push({
      avatarId,
      avatarName: String(value.avatarName ?? "").trim(),
      avatarIcon: String(value.avatarIcon ?? "").trim(),
      avatarIconUrl: String(value.avatarIconUrl ?? "").trim(),
      followerCount: toSafeNumber(value.followerCount),
      followingCount: toSafeNumber(value.followingCount),
      postCount: toSafeNumber(value.postCount),
    });
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
    const mimeType = String(value?.mimeType ?? "").trim().toLowerCase();

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

  const n = Number(value);
  if (!Number.isFinite(n)) {
    return 0;
  }

  return n;
}

function getAnnouncementTokenName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  return String(announcement?.tokenName ?? "").trim();
}

function getAnnouncementProductName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  const productBlueprints = announcement?.productBlueprints;
  if (!Array.isArray(productBlueprints)) return "";

  for (const productBlueprint of productBlueprints) {
    const productName = String(productBlueprint?.productName ?? "").trim();
    if (productName) {
      return productName;
    }
  }

  return "";
}

function getAnnouncementCreatedByName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  return String(
    announcement?.createdByName || announcement?.createdBy || "",
  ).trim();
}

function getAnnouncementUpdatedByName(
  announcement: AnnouncementWithResolvedFields | null,
): string {
  return String(
    announcement?.updatedByName || announcement?.updatedBy || "",
  ).trim();
}

export default function AnnouncementDetailPage() {
  const navigate = useNavigate();
  const { announcementId } = useParams<{ announcementId: string }>();

  const [announcement, setAnnouncement] =
    useState<AnnouncementWithResolvedFields | null>(null);
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

  const resetFormFromAnnouncement = useCallback(
    (source: AnnouncementWithResolvedFields) => {
      setInputPayload({
        title: source.title,
        text: source.content,
        images: [],
      });

      setSelectedAvatarIds(normalizeAvatarIds(source.targetAvatars));
    },
    [],
  );

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
      setAnnouncement(result as AnnouncementWithResolvedFields);
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
        published: announcement.published,
        publishedAt: announcement.publishedAt,
        updatedBy: getUpdatedBy(),
      });

      setAnnouncement(result as AnnouncementWithResolvedFields);
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
        targetAnnouncement = (await updateAnnouncement(announcement.id, {
          title: payload.title,
          content: payload.text,
          targetToken: announcement.targetToken,
          targetAvatars: selectedAvatarIds,
          published: announcement.published,
          publishedAt: announcement.publishedAt,
          updatedBy: getUpdatedBy(),
        })) as AnnouncementWithResolvedFields;
      }

      const result = await markAnnouncementPublished(targetAnnouncement.id, {
        updatedBy: getUpdatedBy(),
      });

      setAnnouncement(result as AnnouncementWithResolvedFields);
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
    if (!announcement) return [];
    return normalizeAvatarIds(announcement.targetAvatars);
  }, [announcement]);

  const visibleSelectedAvatarIds = useMemo(() => {
    if (isEditMode) {
      return normalizeAvatarIds(selectedAvatarIds);
    }

    return targetAvatarIds;
  }, [isEditMode, selectedAvatarIds, targetAvatarIds]);

  const productName = useMemo(() => {
    return getAnnouncementProductName(announcement);
  }, [announcement]);

  const initialImageUrls = useMemo(() => {
    return normalizeAttachmentImageUrls(announcement?.attachmentFiles);
  }, [announcement]);

  const owners = useMemo<SalesOwnerItem[]>(() => {
    const details = normalizeAvatarDetails(announcement?.targetAvatarDetails);
    const detailsByAvatarId = new Map(
      details.map((detail) => [String(detail.avatarId ?? "").trim(), detail]),
    );

    return targetAvatarIds.map((avatarId) => {
      const detail = detailsByAvatarId.get(avatarId);
      const avatarName = String(detail?.avatarName ?? "").trim();
      const avatarIcon = String(
        detail?.avatarIconUrl || detail?.avatarIcon || "",
      ).trim();

      return {
        avatarId,
        avatarName: avatarName || avatarId,
        avatarIconUrl: avatarIcon,
        productName,
        followerCount: toSafeNumber(detail?.followerCount),
        followingCount: toSafeNumber(detail?.followingCount),
        postCount: toSafeNumber(detail?.postCount),
      };
    });
  }, [announcement, productName, targetAvatarIds]);

  const pageTitle = useMemo(() => {
    const tokenName = getAnnouncementTokenName(announcement);
    return tokenName || productName || "告知詳細";
  }, [announcement, productName]);

  const createdByName = getAnnouncementCreatedByName(announcement);
  const updatedByName = getAnnouncementUpdatedByName(announcement);
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
      title={pageTitle}
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
            selectedAvatarIds={visibleSelectedAvatarIds}
            onSelectionChange={isEditMode ? handleSelectionChange : undefined}
          />
        </div>

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