// frontend/console/shell/src/pages/announcementCreatePage.tsx

import { useCallback, useMemo, useState } from "react";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import InputCard from "../features/announcement/presentation/components/inputCard";
import AnnouncementCreateProgressModal from "../features/announcement/presentation/components/announcementCreateProgressModal";

import type { AnnouncementInputPayload } from "../features/announcement/application/announcement_input";
import { useAnnouncementCreatePage } from "../features/announcement/presentation/hook/useAnnouncementCreatePage";

const initialInputPayload: AnnouncementInputPayload = {
  title: "",
  text: "",
  attachments: [],
};

export default function AnnouncementCreatePage() {
  const { vm, state, handlers } = useAnnouncementCreatePage();

  const [inputPayload, setInputPayload] =
    useState<AnnouncementInputPayload>(initialInputPayload);

  const {
    sales,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    owners,
  } = vm;

  const {
    isSaving,
    isSending,
    progress,
    progressOpen,
  } = state;

  const {
    onBack,
    onSaveAnnouncement,
    onSendAnnouncement,
    onCloseProgress,
  } = handlers;

  const targetAvatarIds = useMemo(
    () => owners.map((owner) => owner.avatarId),
    [owners],
  );

  const targetAvatarCount = targetAvatarIds.length;

  const handleInputChange = useCallback(
    (payload: AnnouncementInputPayload) => {
      setInputPayload(payload);
    },
    [],
  );

  const handleSave = useCallback(async () => {
    if (isSaving || isSending) {
      return;
    }

    try {
      await onSaveAnnouncement({
        payload: inputPayload,
        targetAvatarIds,
      });
    } catch (error) {
      console.error(
        "[AnnouncementCreatePage] save announcement failed",
        error,
      );
    }
  }, [
    inputPayload,
    isSaving,
    isSending,
    onSaveAnnouncement,
    targetAvatarIds,
  ]);

  const handleSend = useCallback(async () => {
    if (isSaving || isSending) {
      return;
    }

    try {
      await onSendAnnouncement({
        payload: inputPayload,
        targetAvatarIds,
      });
    } catch (error) {
      console.error(
        "[AnnouncementCreatePage] send announcement failed",
        error,
      );
    }
  }, [
    inputPayload,
    isSaving,
    isSending,
    onSendAnnouncement,
    targetAvatarIds,
  ]);

  if (!sales) {
    return (
      <PageStyle
        layout="single"
        title="告知を作成"
        onBack={onBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          表示可能な告知作成情報がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="告知を作成"
        onBack={onBack}
        onSave={handleSave}
        isSaving={isSaving}
        onSend={handleSend}
        isSending={isSending}
      >
        <div className="space-y-4">
          <InputCard
            title="入力"
            saving={isSaving}
            sending={isSending}
            onChange={handleInputChange}
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
          isSaving ||
          isSending ||
          progress.isBlockingNavigation
            ? undefined
            : onCloseProgress
        }
      />
    </>
  );
}