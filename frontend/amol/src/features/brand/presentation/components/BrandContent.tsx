// frontend/amol/src/features/brand/presentation/components/BrandContent.tsx

import type {
  BrandDetail,
  BrandListItem,
} from "../../../shared/types/brand";

import BrandBackground from "./BrandBackground";
import BrandIcon from "./BrandIcon";
import BrandListSection from "./BrandListSection";
import BrandWebsiteLink from "./BrandWebsiteLink";

type BrandContentProps = {
  brand: BrandDetail;
  listItems: BrandListItem[];
};

export default function BrandContent({
  brand,
  listItems,
}: BrandContentProps) {
  const brandName =
    brand.brandName.trim();

  const companyName =
    brand.companyName.trim();

  const description =
    brand.description.trim();

  const websiteUrl =
    brand.websiteUrl.trim();

  return (
    <div className="brand-page">
      <BrandBackground
        brand={brand}
      />

      <section className="brand-page-profile">
        <BrandIcon
          brand={brand}
        />

        <div className="brand-page-profile-body">
          <h1>
            {brandName ||
              "名称未設定のブランド"}
          </h1>

          {companyName ? (
            <p className="brand-page-company">
              {companyName}
            </p>
          ) : null}

          {websiteUrl ? (
            <BrandWebsiteLink
              url={websiteUrl}
            />
          ) : null}
        </div>
      </section>

      {description ? (
        <section className="brand-page-section">
          <h2>説明</h2>

          <p className="brand-page-description">
            {description}
          </p>
        </section>
      ) : null}

      <BrandListSection
        listIds={brand.listIds}
        listItems={listItems}
      />
    </div>
  );
}