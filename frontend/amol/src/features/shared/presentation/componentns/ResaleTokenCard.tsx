// frontend/amol/src/features/shared/presentation/componentns/ResaleTokenCard.tsx

import MediaIcon from "../../../../components/ui/MediaIcon";

export type ResaleTokenCardProps = {
  brandName?: string | null;
  tokenName?: string | null;
  tokenIcon?: string | null;
};

export default function ResaleTokenCard({
  brandName,
  tokenName,
  tokenIcon,
}: ResaleTokenCardProps) {
  const safeBrandName = brandName?.trim() || "ブランド名未設定";
  const safeTokenName = tokenName?.trim() || "トークン名未設定";
  const safeTokenIcon = tokenIcon?.trim() || "";

  if (!safeTokenIcon && !tokenName?.trim()) {
    return null;
  }

  return (
    <div className="resale-product-detail__token">
      <MediaIcon
        src={safeTokenIcon}
        alt={tokenName?.trim() ? `${tokenName.trim()}のトークンアイコン` : "トークンアイコン"}
        fallback="◎"
        size="lg"
        shape="rounded"
        className="resale-product-detail__token-icon"
      />

      <div className="resale-product-detail__token-body">
        <span className="resale-product-detail__token-label">
          {safeBrandName}
        </span>

        <span className="resale-product-detail__token-name">
          {safeTokenName}
        </span>
      </div>
    </div>
  );
}