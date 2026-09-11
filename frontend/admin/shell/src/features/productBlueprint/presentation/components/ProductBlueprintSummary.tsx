// frontend/admin/shell/src/features/company/presentation/components/ProductBlueprintSummary.tsx

import type { ContractProductBlueprintDetail } from "../../model/productBlueprintDetail";

const CATEGORY_FIELD_LABELS: Record<string, string> = {
  weight: "重量",
  vintage: "ヴィンテージ",
  region: "地域・産地",
  fit: "フィット",
  material: "素材",
  washTags: "洗濯表示",
  alcoholContent: "アルコール度数",
};

type ProductBlueprintSummaryProps = {
  productBlueprint: ContractProductBlueprintDetail;
};

function formatCategoryFields(
  categoryFields: Record<string, unknown>,
  fieldLabels: Record<string, string>,
): string {
  const entries = Object.entries(categoryFields);
  if (entries.length === 0) return "-";

  return entries
    .map(([key, value]) => {
      const label = fieldLabels[key] ?? key;
      if (value == null) return `${label}: -`;
      if (typeof value === "object") return `${label}: ${JSON.stringify(value)}`;
      return `${label}: ${String(value)}`;
    })
    .join("\n");
}

export default function ProductBlueprintSummary({
  productBlueprint,
}: ProductBlueprintSummaryProps) {
  return (
    <section className="ui-detail-section">
      <dl className="ui-detail-definition-list">
        <dd>
          {productBlueprint.productBlueprintCategoryPath.length > 0
            ? productBlueprint.productBlueprintCategoryPath.join(" / ")
            : "-"}
        </dd>
        <dd style={{ whiteSpace: "pre-wrap" }}>
          {formatCategoryFields(
            productBlueprint.categoryFields,
            CATEGORY_FIELD_LABELS,
          )}
        </dd>
      </dl>
    </section>
  );
}
