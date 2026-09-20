// frontend/mall/src/features/brand/presentation/components/BrandContent.tsx

import SectionHeader from "../../../../components/ui/SectionHeader";
import ReportFlagButton from "../../../shared/presentation/components/ReportFlagButton";
import type { BrandDetail } from "../../../shared/types/brand";
import type { MallListItem } from "../../../shared/types/list";

import BrandBackground from "./BrandBackground";
import BrandIcon from "./BrandIcon";
import BrandListSection from "./BrandListSection";
import BrandWebsiteLink from "./BrandWebsiteLink";

type BrandContentProps = {
  brand: BrandDetail;
  listItems: MallListItem[];
  canReport: boolean;
  onReport: () => void;
};

export default function BrandContent({
  brand,
  listItems,
  canReport,
  onReport,
}: BrandContentProps) {
  const brandName = brand.brandName.trim();
  const companyName = brand.companyName.trim();
  const description = brand.description.trim();
  const websiteUrl = brand.websiteUrl.trim();

  return (
    <div className="brand-page">
      <BrandBackground brand={brand} />

      <section className="brand-page-profile">
        <BrandIcon brand={brand} />

        <div className="brand-page-profile-body">
          <h1>{brandName || "名称未設定のブランド"}</h1>

          {companyName ? <p className="brand-page-company">{companyName}</p> : null}

          {websiteUrl ? <BrandWebsiteLink url={websiteUrl} /> : null}
        </div>
      </section>

      {description ? (
        <section className="brand-page-section">
          <SectionHeader
            title="説明"
            titleAs="h2"
            right={
              <ReportFlagButton
                label="ブランドを通報"
                disabled={!canReport}
                onClick={onReport}
              />
            }
          />

          <p className="brand-page-description">{description}</p>
        </section>
      ) : null}

      <BrandListSection listIds={brand.listIds} listItems={listItems} />
    </div>
  );
}