// frontend/console/shell/src/pages/announcementCreatePage.tsx

import {
  useCallback,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
} from "react-router-dom";

import PageStyle from "../layout/PageStyle/PageStyle";
import AdminCard from "../features/admin/presentation/components/AdminCard";
import LogCard from "../features/log/presentation/LogCard";
import InputCard from "../features/announcement/presentation/components/inputCard";

import type { SubmitPayload } from "../features/announcement/presentation/components/inputCard";

import {
  useAnnouncementCreatePage,
  type AnnouncementCreateInputPayload,
} from "../features/announcement/presentation/hook/useAnnouncementCreatePage";

const initialInputPayload: AnnouncementCreateInputPayload = {
  title: "",
  text: "",
  images: [],
};

export default function AnnouncementCreatePage() {
  const navigate = useNavigate();

  const { vm, handlers } =
    useAnnouncementCreatePage();

  const [inputPayload, setInputPayload] =
    useState<AnnouncementCreateInputPayload>(
      initialInputPayload,
    );

  const [isSavingInput, setIsSavingInput] =
    useState(false);

  const [isSendingInput, setIsSendingInput] =
    useState(false);

  const {
    sales,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    owners,
  } = vm;

  const {
    onBack,
    onSaveAnnouncement,
    onSendAnnouncement,
  } = handlers;

  const targetAvatarIds = useMemo(
    () =>
      owners.map(
        (owner) => owner.avatarId,
      ),
    [owners],
  );

  const targetAvatarCount =
    targetAvatarIds.length;

  const handleInputChange = useCallback(
    (payload: SubmitPayload) => {
      setInputPayload({
        title: payload.title,
        text: payload.text,
        images: payload.images,
      });
    },
    [],
  );

  const buildSubmitPayload =
    useCallback(
      (): AnnouncementCreateInputPayload => {
        return {
          title: inputPayload.title.trim(),
          text: inputPayload.text.trim(),
          images: inputPayload.images,
        };
      },
      [inputPayload],
    );

  const handleSave =
    useCallback(async () => {
      if (
        isSavingInput ||
        isSendingInput
      ) {
        return;
      }

      setIsSavingInput(true);

      try {
        await onSaveAnnouncement({
          payload:
            buildSubmitPayload(),
          targetAvatarIds,
        });

        window.alert(
          "告知を保存しました。",
        );

        navigate(
          "/sales",
        );
      } catch (error) {
        console.error(
          "[AnnouncementCreatePage] save announcement failed",
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
      buildSubmitPayload,
      isSavingInput,
      isSendingInput,
      navigate,
      onSaveAnnouncement,
      targetAvatarIds,
    ]);

  const handleSend =
    useCallback(async () => {
      if (
        isSavingInput ||
        isSendingInput
      ) {
        return;
      }

      setIsSendingInput(true);

      try {
        await onSendAnnouncement({
          payload:
            buildSubmitPayload(),
          targetAvatarIds,
        });

        window.alert(
          "告知を送信しました。",
        );
      } catch (error) {
        console.error(
          "[AnnouncementCreatePage] send announcement failed",
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
      buildSubmitPayload,
      isSavingInput,
      isSendingInput,
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
    <PageStyle
      layout="grid-2"
      title="告知を作成"
      onBack={onBack}
      onSave={handleSave}
      isSaving={isSavingInput}
      onSend={handleSend}
      isSending={isSendingInput}
    >
      <div className="space-y-4">
        <InputCard
          title="入力"
          saving={isSavingInput}
          sending={isSendingInput}
          onChange={handleInputChange}
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