// frontend/amol/src/features/catalog/components/ProductInfoCard.tsx
import InfoList, { InfoRow } from "../../../components/ui/InfoList";
import SectionCard from "../../../components/ui/SectionCard";
import SectionHeader from "../../../components/ui/SectionHeader";
import Tab from "../../../components/ui/Tab";
import type { CatalogProductBlueprint } from "../types";

type ProductInfoCardProps = {
  productBlueprint: CatalogProductBlueprint;
};

export default function ProductInfoCard({
  productBlueprint,
}: ProductInfoCardProps) {
  return (
    <SectionCard className="catalog-page-card">
      <SectionHeader
        title="商品情報"
        titleAs="h2"
        className="catalog-page-card-header"
      />

      <InfoList className="catalog-page-definition-list">
        <InfoRow label="商品名">{productBlueprint.productName}</InfoRow>
        <InfoRow label="ブランド">{productBlueprint.brandName}</InfoRow>
        <InfoRow label="会社名">{productBlueprint.companyName}</InfoRow>
        <InfoRow label="カテゴリ">{productBlueprint.itemType}</InfoRow>
        <InfoRow label="フィット">{productBlueprint.fit}</InfoRow>
        <InfoRow label="素材">{productBlueprint.material}</InfoRow>
        <InfoRow label="重量">{productBlueprint.weight}g</InfoRow>
        <InfoRow label="商品IDタグ">
          {productBlueprint.productIdTagType}
        </InfoRow>
      </InfoList>

      {productBlueprint.qualityAssurance.length > 0 ? (
        <div className="catalog-page-chip-list">
          {productBlueprint.qualityAssurance.map((item) => (
            <Tab key={item} className="catalog-page-chip" disabled>
              {item}
            </Tab>
          ))}
        </div>
      ) : null}
    </SectionCard>
  );
}