// frontend/mall/src/features/catalog/presentation/components/ProductInfoCard.tsx

import Chip from "../../../../components/ui/Chip";
import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import TextButton from "../../../../components/ui/TextButton";
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
    <SectionCard className="catalog-page-card">
      <SectionHeader
        title="商品情報"
        titleAs="h2"
        className="catalog-page-card-header"
      />

      <InfoList className="catalog-page-definition-list">
        {viewModel.rows.map((row) => {
          const isBrandRow = row.label === "ブランド";

          return (
            <InfoRow key={row.key} label={row.label}>
              {isBrandRow ? (
                <TextButton
                  className="catalog-page-brand-link"
                  onClick={onBrandClick}
                  disabled={!onBrandClick}
                  aria-label={`${productBlueprint.brandName}のブランドページへ移動`}
                >
                  {row.value}
                </TextButton>
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
    </SectionCard>
  );
}