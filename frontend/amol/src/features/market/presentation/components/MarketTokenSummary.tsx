// frontend/amol/src/features/market/presentation/components/MarketTokenSummary.tsx

type MarketTokenSummaryProps = {
  tokenName: string;
  tokenIcon: string;
  brandName: string;
};

export default function MarketTokenSummary({
  tokenName,
  tokenIcon,
  brandName,
}: MarketTokenSummaryProps) {
  if (!tokenName && !tokenIcon) {
    return null;
  }

  return (
    <div className="market-detail-page__token">
      {tokenIcon ? (
        <img
          src={tokenIcon}
          alt={tokenName || "トークンアイコン"}
          className="market-detail-page__token-icon"
        />
      ) : null}

      <div className="market-detail-page__token-body">
        <span className="market-detail-page__token-label">
          {brandName || "ブランド名未設定"}
        </span>

        <span className="market-detail-page__token-name">
          {tokenName || "トークン名未設定"}
        </span>
      </div>
    </div>
  );
}