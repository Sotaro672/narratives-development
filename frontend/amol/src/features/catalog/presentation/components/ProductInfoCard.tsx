// frontend/amol/src/features/catalog/presentation/components/ProductInfoCard.tsx

import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Tab from "../../../../components/ui/Tab";
import type { CatalogProductBlueprint } from "../../types";

type ProductCategoryKind = "apparel" | "alcohol" | "unknown";

type ProductInfoCardProps = {
  productBlueprint: CatalogProductBlueprint;
  categoryKind?: ProductCategoryKind;
};

type ProductBlueprintExtraFields = CatalogProductBlueprint & {
  category?: string | null;
  categoryCode?: string | null;
  classification?: string | null;
  region?: string | null;
  vintage?: string | number | null;
  alcoholContent?: string | number | null;
};

function isNonEmptyText(value: unknown): value is string {
  return typeof value === "string" && value.trim() !== "";
}

function formatNullableText(value: unknown): string {
  if (typeof value === "string") {
    return value.trim();
  }

  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value);
  }

  return "";
}

function formatWeight(value: unknown): string {
  if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
    return "";
  }

  return `${value}g`;
}

function formatAlcoholContent(value: unknown): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return `${value}%`;
  }

  if (typeof value === "string") {
    const text = value.trim();

    if (!text) {
      return "";
    }

    return text.includes("%") ? text : `${text}%`;
  }

  return "";
}

function resolveCategoryLabel(
  productBlueprint: ProductBlueprintExtraFields,
): string {
  return (
    formatNullableText(productBlueprint.category) ||
    formatNullableText(productBlueprint.categoryCode) ||
    formatNullableText(productBlueprint.classification) ||
    formatNullableText(productBlueprint.itemType)
  );
}

function resolveQualityAssuranceItems(
  qualityAssurance: CatalogProductBlueprint["qualityAssurance"],
): string[] {
  if (Array.isArray(qualityAssurance)) {
    return qualityAssurance
      .map((item) => {
        if (typeof item === "string") {
          return item.trim();
        }

        if (
          item &&
          typeof item === "object" &&
          "label" in item &&
          typeof item.label === "string"
        ) {
          return item.label.trim();
        }

        if (
          item &&
          typeof item === "object" &&
          "title" in item &&
          typeof item.title === "string"
        ) {
          return item.title.trim();
        }

        return "";
      })
      .filter(Boolean);
  }

  if (typeof qualityAssurance === "string") {
    const text = qualityAssurance.trim();
    return text ? [text] : [];
  }

  return [];
}

function renderOptionalRow(label: string, value: unknown) {
  const text = formatNullableText(value);

  if (!text) {
    return null;
  }

  return <InfoRow label={label}>{text}</InfoRow>;
}

export default function ProductInfoCard({
  productBlueprint,
  categoryKind = "unknown",
}: ProductInfoCardProps) {
  const product = productBlueprint as ProductBlueprintExtraFields;

  const categoryLabel = resolveCategoryLabel(product);
  const qualityAssuranceItems = resolveQualityAssuranceItems(
    productBlueprint.qualityAssurance,
  );

  const isAlcohol = categoryKind === "alcohol";
  const isApparel = categoryKind === "apparel" || categoryKind === "unknown";

  const weightLabel = formatWeight(product.weight);
  const alcoholContentLabel = formatAlcoholContent(product.alcoholContent);

  return (
    <SectionCard className="catalog-page-card">
      <SectionHeader
        title="商品情報"
        titleAs="h2"
        className="catalog-page-card-header"
      />

      <InfoList className="catalog-page-definition-list">
        <InfoRow label="商品名">{product.productName}</InfoRow>
        <InfoRow label="ブランド">{product.brandName}</InfoRow>
        <InfoRow label="会社名">{product.companyName}</InfoRow>

        {categoryLabel ? (
          <InfoRow label="カテゴリ">{categoryLabel}</InfoRow>
        ) : null}

        {isAlcohol ? (
          <>
            {renderOptionalRow("原材料", product.material)}
            {renderOptionalRow("産地", product.region)}
            {renderOptionalRow("ヴィンテージ", product.vintage)}

            {alcoholContentLabel ? (
              <InfoRow label="アルコール度数">{alcoholContentLabel}</InfoRow>
            ) : null}
          </>
        ) : null}

        {isApparel ? (
          <>
            {renderOptionalRow("フィット", product.fit)}
            {renderOptionalRow("素材", product.material)}

            {weightLabel ? (
              <InfoRow label="重量">{weightLabel}</InfoRow>
            ) : null}
          </>
        ) : null}

        {isNonEmptyText(product.productIdTagType) ? (
          <InfoRow label="商品IDタグ">{product.productIdTagType}</InfoRow>
        ) : null}
      </InfoList>

      {qualityAssuranceItems.length > 0 ? (
        <div className="catalog-page-chip-list">
          {qualityAssuranceItems.map((item) => (
            <Tab key={item} className="catalog-page-chip" disabled>
              {item}
            </Tab>
          ))}
        </div>
      ) : null}
    </SectionCard>
  );
}