// frontend/mall/src/features/avatar/components/AvatarCreateProgressModal.tsx

import ProgressModal from "../../shared/presentation/components/ProgressModal";
import type { AvatarCreateProgress } from "../models/avatarCreateProgress";

export type AvatarCreateProgressModalProps = {
  open: boolean;
  progress: AvatarCreateProgress;
  onClose?: () => void;
};

export default function AvatarCreateProgressModal({
  open,
  progress,
  onClose,
}: AvatarCreateProgressModalProps) {
  return (
    <ProgressModal
      open={open}
      progress={progress}
      onClose={onClose}
      idPrefix="avatar-create-progress-modal"
      progressAriaLabel="アバターアイコンの転送進捗"
      savingNotice="画像転送は完了しています。アバター情報の保存処理を続けています。"
    />
  );
}