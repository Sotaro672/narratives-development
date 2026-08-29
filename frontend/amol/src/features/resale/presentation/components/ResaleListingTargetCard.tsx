// frontend/amol/src/features/resale/presentation/components/ResaleListingTargetCard.tsx

import MediaIcon from "../../../../components/ui/MediaIcon";
import SectionHeader from "../../../../components/ui/SectionHeader";
import type { ResaleCreateTarget } from "../types/resaleCreatePageTypes";

export type ResaleListingTargetCardTarget = Pick<
  ResaleCreateTarget,
  "brandName" | "productName" | "tokenIconUrl" | "tokenName"
>;

export type ResaleListingTargetCardProps = {
  target: ResaleListingTargetCardTarget;
};

export default function ResaleListingTargetCard({ target }: ResaleListingTargetCardProps) {
  const { brandName, productName, tokenIconUrl, tokenName } = target;

  return (
    <section className="page-card">
      <SectionHeader title="出品対象" titleAs="h2" />

      <div className="resale-token-summary">
        <MediaIcon
          src={tokenIconUrl}
          alt={tokenName ? `${tokenName}のトークンアイコン` : "トークンアイコン"}
          fallback="◎"
          size="lg"
          shape="rounded"
          className="resale-token-summary__icon"
        />

        <div className="resale-token-summary__body">
          <p className="resale-token-summary__token-name">{tokenName || "-"}</p>
          <p className="resale-token-summary__brand-name">{brandName || "-"}</p>
          {productName ? <p className="resale-token-summary__product-name">{productName}</p> : null}
        </div>
      </div>
    </section>
  );
}