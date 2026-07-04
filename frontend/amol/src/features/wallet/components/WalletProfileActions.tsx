// frontend\amol\src\features\wallet\components\WalletProfileActions.tsx
import { useNavigate } from "react-router-dom";

import Button from "../../../components/ui/Button";

type WalletProfileActionsProps = {
  avatarId: string;
  isOwnAvatar: boolean;
};

export default function WalletProfileActions({
  isOwnAvatar,
}: WalletProfileActionsProps) {
  const navigate = useNavigate();

  if (!isOwnAvatar) {
    return null;
  }

  return (
    <div className="wallet-page-profile-actions-bar">
      <div className="wallet-page-profile-actions-bar__inner">
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => navigate("/avatar")}
        >
          アバター編集
        </Button>
      </div>
    </div>
  );
}