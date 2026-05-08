// frontend/amol/src/features/wallet/components/WalletProfileActions.tsx
import { useNavigate } from "react-router-dom";

import Button from "../../../components/ui/Button";

type WalletProfileActionsProps = {
  avatarId: string;
  isOwnAvatar: boolean;
  isFollowing?: boolean;
  followPosting?: boolean;
  onFollowClick?: () => void | Promise<void>;
};

export default function WalletProfileActions({
  avatarId,
  isOwnAvatar,
  isFollowing = false,
  followPosting = false,
  onFollowClick,
}: WalletProfileActionsProps) {
  const navigate = useNavigate();

  const handleShareProfile = () => {
    if (!avatarId) {
      window.alert("アバターIDを取得できませんでした。");
      return;
    }

    navigate(`/avatars/${avatarId}/share-qr`);
  };

  const handleFollow = async () => {
    if (!onFollowClick || followPosting || isFollowing) {
      return;
    }

    try {
      await onFollowClick();
    } catch (error) {
      console.error(error);
      window.alert("フォローに失敗しました。");
    }
  };

  return (
    <div className="wallet-page-profile-actions-bar">
      <div className="wallet-page-profile-actions-bar__inner">
        {isOwnAvatar ? (
          <>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => navigate("/avatar")}
            >
              アバター編集
            </Button>

            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={handleShareProfile}
            >
              プロフィールをシェア
            </Button>
          </>
        ) : (
          <Button
            type="button"
            variant="primary"
            size="sm"
            onClick={handleFollow}
            disabled={followPosting || isFollowing}
          >
            {isFollowing ? "フォロー中" : followPosting ? "フォロー中..." : "フォロー"}
          </Button>
        )}
      </div>
    </div>
  );
}