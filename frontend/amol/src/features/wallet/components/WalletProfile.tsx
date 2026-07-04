// frontend/amol/src/features/wallet/components/WalletProfile.tsx

type WalletProfileProps = {
  avatarName: string;
  avatarIcon: string;
  profile: string;
  isOwnAvatar: boolean;
};

export default function WalletProfile({
  avatarName,
  avatarIcon,
  profile,
  isOwnAvatar,
}: WalletProfileProps) {
  const shouldShowProfile = !isOwnAvatar && Boolean(profile);

  return (
    <div className="wallet-page-profile">
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
        {avatarName ? (
          <div className="wallet-page-profile__name">{avatarName}</div>
        ) : null}

        {shouldShowProfile ? (
          <p className="wallet-page-profile__text">{profile}</p>
        ) : null}
      </div>
    </div>
  );
}