// frontend/amol/src/features/wallet/components/WalletTokenContentsCard.tsx

type WalletTokenContentsCardProps = {
  tokenIconUrl?: string | null;
  tokenName?: string;
  productName?: string;
  onClick?: () => void;
};

export default function WalletTokenContentsCard({
  tokenIconUrl,
  tokenName,
  productName,
  onClick,
}: WalletTokenContentsCardProps) {
  return (
    <button
      type="button"
      className="wallet-token-card"
      onClick={onClick}
      aria-label={`${tokenName || "トークン"}の詳細を開く`}
    >
      <div className="wallet-token-card__icon-wrap">
        {tokenIconUrl ? (
          <img
            src={tokenIconUrl}
            alt={tokenName || "トークンアイコン"}
            className="wallet-token-card__icon"
            loading="lazy"
          />
        ) : (
          <div className="wallet-token-card__icon wallet-token-card__icon--fallback">
            ◎
          </div>
        )}
      </div>

      <div className="wallet-token-card__body">
        <p className="wallet-token-card__name">
          {tokenName || "名称未設定のトークン"}
        </p>

        {productName ? (
          <p className="wallet-token-card__product-name">{productName}</p>
        ) : null}
      </div>
    </button>
  );
}