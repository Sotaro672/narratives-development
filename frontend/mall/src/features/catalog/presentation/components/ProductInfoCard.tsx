// frontend/mall/src/features/catalog/presentation/components/ProductInfoCard.tsx

import Chip from "../../../../components/ui/Chip";
import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import TextLink from "../../../../components/ui/textLink";
import type { CatalogProductBlueprint } from "../../../shared/types/catalog";
import {
  createProductInfoCardViewModel,
  type ProductCategoryKind,
} from "../../application/catalogProductInfoViewModelFactory";

type ProductInfoCardProps = {
  productBlueprint: CatalogProductBlueprint;
  categoryKind?: ProductCategoryKind;
  onBrandClick?: () => void;
};

export default function ProductInfoCard({
  productBlueprint,
  categoryKind = "unknown",
  onBrandClick,
}: ProductInfoCardProps) {
  const viewModel = createProductInfoCardViewModel({
    productBlueprint,
    categoryKind,
  });

  return (
    <section className="catalog-page-product-info">
      <InfoList>
        {viewModel.rows.map((row) => {
          const isBrandRow = row.label === "ブランド";

          return (
            <InfoRow key={row.key} label={row.label}>
              {isBrandRow ? (
                <TextLink
                  onClick={onBrandClick}
                  disabled={!onBrandClick}
                  aria-label={`${productBlueprint.brandName}のブランドページへ移動`}
                >
                  {row.value}
                </TextLink>
              ) : (
                row.value
              )}
            </InfoRow>
          );
        })}
      </InfoList>

      {viewModel.qualityAssuranceItems.length > 0 ? (
        <div className="catalog-page-chip-list">
          {viewModel.qualityAssuranceItems.map((item) => (
            <Chip
              key={item}
              className="catalog-page-chip"
              size="sm"
              disabled
            >
              {item}
            </Chip>
          ))}
        </div>
      ) : null}
    </section>
  );
}