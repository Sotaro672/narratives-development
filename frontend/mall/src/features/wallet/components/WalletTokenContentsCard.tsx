// frontend/amol/src/features/wallet/components/WalletTokenContentsCard.tsx

import Media from "../../../components/ui/Media";

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
          <Media
            src={tokenIconUrl}
            alt={tokenName || "トークンアイコン"}
            loading="lazy"
            fit="cover"
          />
        ) : (
          <div className="ui-media-fallback wallet-token-card__icon--fallback">
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