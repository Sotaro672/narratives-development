// frontend/amol/src/features/resale/presentation/components/ResaleListingTargetCard.tsx

import MediaIcon from "../../../../components/ui/MediaIcon";
import type { ResaleCreateTarget } from "../types/resaleCreatePageTypes";

export type ResaleListingTargetCardTarget = Pick<
  ResaleCreateTarget,
  "brandName" | "productName" | "tokenIconUrl" | "tokenName"
>;

export type ResaleListingTargetCardProps = {
  target: ResaleListingTargetCardTarget;
};

export default function ResaleListingTargetCard({
  target,
}: ResaleListingTargetCardProps) {
  const { brandName, productName, tokenIconUrl, tokenName } = target;

  return (
    <>
      <p className="resale-detail-page__brand">{brandName || "ブランド名未設定"}</p>

      <h1 className="resale-detail-page__title">
        {productName || tokenName || "商品名未設定"}
      </h1>

      <div className="resale-detail-page__token">
        <MediaIcon
          src={tokenIconUrl}
          alt={tokenName ? `${tokenName}のトークンアイコン` : "トークンアイコン"}
          fallback="◎"
          size="lg"
          shape="rounded"
          className="resale-detail-page__token-icon"
        />

        <div className="resale-detail-page__token-body">
          <span className="resale-detail-page__token-label">トークン</span>
          <span className="resale-detail-page__token-name">
            {tokenName || "トークン名未設定"}
          </span>
        </div>
      </div>
    </>
  );
}