// frontend/amol/src/features/market/presentation/components/MarketSellerCard.tsx

type MarketSellerCardProps = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  onOpen: () => void;
};

export default function MarketSellerCard({
  avatarId,
  avatarName,
  avatarIcon,
  onOpen,
}: MarketSellerCardProps) {
  if (
    !avatarId &&
    !avatarName &&
    !avatarIcon
  ) {
    return null;
  }

  return (
    <button
      type="button"
      className="market-detail-page__seller market-detail-page__seller--button"
      onClick={onOpen}
      disabled={!avatarId}
    >
      {avatarIcon ? (
        <img
          src={avatarIcon}
          alt={
            avatarName ||
            "出品者アイコン"
          }
          className="market-detail-page__seller-icon"
        />
      ) : (
        <span
          className="market-detail-page__seller-icon market-detail-page__seller-icon--placeholder"
          aria-hidden="true"
        >
          ◎
        </span>
      )}

      <div className="market-detail-page__seller-body">
        <span className="market-detail-page__seller-label">
          出品者
        </span>

        <span className="market-detail-page__seller-name">
          {avatarName ||
            avatarId ||
            "アバター名未設定"}
        </span>
      </div>

      {avatarId ? (
        <span
          className="market-detail-page__seller-arrow"
          aria-hidden="true"
        >
          ›
        </span>
      ) : null}
    </button>
  );
}