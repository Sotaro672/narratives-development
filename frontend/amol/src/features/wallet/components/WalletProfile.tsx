// frontend/amol/src/features/wallet/components/WalletProfile.tsx
import { useNavigate } from "react-router-dom";

type WalletProfileProps = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  profile: string;
  followerCount: number;
  followingCount: number;
  isOwnAvatar: boolean;
};

export default function WalletProfile({
  avatarId,
  avatarName,
  avatarIcon,
  profile,
  followerCount,
  followingCount,
  isOwnAvatar,
}: WalletProfileProps) {
  const navigate = useNavigate();

  const shouldShowStats = isOwnAvatar;
  const shouldShowProfile = !isOwnAvatar && Boolean(profile);

  const handleOpenFollowPage = () => {
    if (!isOwnAvatar) {
      return;
    }

    if (!avatarId) {
      window.alert("アバターIDを取得できませんでした。");
      return;
    }

    navigate(`/avatars/${encodeURIComponent(avatarId)}/follow?tab=following`);
  };

  const profileContent = (
    <>
      <div className="wallet-page-profile__avatar-area">
        <div className="wallet-page-profile__avatar-wrap">
          {avatarIcon ? (
            <img
              src={avatarIcon}
              alt={avatarName || "アバター画像"}
              className="wallet-page-profile__avatar"
            />
          ) : (
            <div className="wallet-page-profile__avatar wallet-page-profile__avatar--fallback">
              👤
            </div>
          )}
        </div>
      </div>

      <div className="wallet-page-profile__body">
        {shouldShowStats ? (
          <div className="wallet-page-profile__stats">
            <div className="wallet-page-profile__stat">
              <span className="wallet-page-profile__stat-value">
                {followerCount}
              </span>
              <span className="wallet-page-profile__stat-label">
                フォロワー
              </span>
            </div>

            <div className="wallet-page-profile__stat">
              <span className="wallet-page-profile__stat-value">
                {followingCount}
              </span>
              <span className="wallet-page-profile__stat-label">
                フォロー
              </span>
            </div>
          </div>
        ) : null}

        {shouldShowProfile ? (
          <p className="wallet-page-profile__text">{profile}</p>
        ) : null}
      </div>
    </>
  );

  if (isOwnAvatar) {
    return (
      <button
        type="button"
        className="wallet-page-profile wallet-page-profile--button"
        onClick={handleOpenFollowPage}
      >
        {profileContent}
      </button>
    );
  }

  return <div className="wallet-page-profile">{profileContent}</div>;
}