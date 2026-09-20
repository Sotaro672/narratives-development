// frontend/mall/src/features/resale/presentation/components/ResaleCreateProgressModal.tsx

import ProgressModal from "../../../shared/presentation/components/ProgressModal";
import type { ResaleCreateProgress } from "../models/resaleCreateProgress";

export type ResaleCreateProgressModalProps = {
  open: boolean;
  progress: ResaleCreateProgress;
  onClose?: () => void;
};

export default function ResaleCreateProgressModal({
  open,
  progress,
  onClose,
}: ResaleCreateProgressModalProps) {
  return (
    <ProgressModal
      open={open}
      progress={progress}
      onClose={onClose}
      idPrefix="resale-create-progress-modal"
      progressAriaLabel="商品状態画像の転送進捗"
      savingNotice="画像転送は完了しています。出品情報の保存処理を続けています。"
    />
  );
}