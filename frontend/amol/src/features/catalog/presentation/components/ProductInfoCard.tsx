// frontend/amol/src/features/catalog/presentation/components/ProductInfoCard.tsx

import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Tab from "../../../../components/ui/Tab";
import {
  createProductInfoCardViewModel,
  type ProductCategoryKind,
} from "../../application/catalogProductInfoViewModelFactory";
import type { CatalogProductBlueprint } from "../../types";

type ProductInfoCardProps = {
  productBlueprint: CatalogProductBlueprint;
  categoryKind?: ProductCategoryKind;
};

export default function ProductInfoCard({
  productBlueprint,
  categoryKind = "unknown",
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
        {viewModel.rows.map((row) => (
          <InfoRow key={row.key} label={row.label}>
            {row.value}
          </InfoRow>
        ))}
      </InfoList>

      {viewModel.qualityAssuranceItems.length > 0 ? (
        <div className="catalog-page-chip-list">
          {viewModel.qualityAssuranceItems.map((item) => (
            <Tab key={item} className="catalog-page-chip" disabled>
              {item}
            </Tab>
          ))}
        </div>
      ) : null}
    </SectionCard>
  );
}