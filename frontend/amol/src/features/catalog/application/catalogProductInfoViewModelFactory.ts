// frontend/amol/src/features/catalog/application/catalogProductInfoViewModelFactory.ts

import type { CatalogProductBlueprint } from "../types";
import {
  formatAlcoholContent,
  formatNullableText,
  resolveCategoryLabel,
  resolveQualityAssuranceItems,
  type ProductCategoryKind,
} from "./catalogProductInfoMapper";

export type ProductInfoRowViewModel = {
  key: string;
  label: string;
  value: string;
};

export type ProductInfoCardViewModel = {
  rows: ProductInfoRowViewModel[];
  qualityAssuranceItems: string[];
};

type ProductInfoDisplayFields = CatalogProductBlueprint & {
  qualityAssurance?: unknown;
};

function createRow(
  key: string,
  label: string,
  value: unknown,
): ProductInfoRowViewModel | null {
  const text = formatNullableText(value);

  if (!text) {
    return null;
  }

  return {
    key,
    label,
    value: text,
  };
}

function createFormattedRow(
  key: string,
  label: string,
  value: string,
): ProductInfoRowViewModel | null {
  if (!value) {
    return null;
  }

  return {
    key,
    label,
    value,
  };
}

function appendRow(
  rows: ProductInfoRowViewModel[],
  row: ProductInfoRowViewModel | null,
): void {
  if (row) {
    rows.push(row);
  }
}

function getCategoryFieldValue(
  categoryFields: Record<string, unknown> | null | undefined,
  key: string,
): unknown {
  if (!categoryFields || !key) {
    return "";
  }

  return categoryFields[key];
}

export function createProductInfoCardViewModel(args: {
  productBlueprint: CatalogProductBlueprint;
  categoryKind?: ProductCategoryKind;
}): ProductInfoCardViewModel {
  const product = args.productBlueprint as ProductInfoDisplayFields;

  const categoryKind =
    args.categoryKind ??
    product.productBlueprintCategoryKind ??
    "unknown";

  const isAlcohol = categoryKind === "alcohol";
  const isApparel = categoryKind === "apparel" || categoryKind === "unknown";

  const rows: ProductInfoRowViewModel[] = [];
  const categoryFields = product.categoryFields ?? null;

  appendRow(rows, createRow("productName", "商品名", product.productName));
  appendRow(rows, createRow("brandName", "ブランド", product.brandName));
  appendRow(rows, createRow("companyName", "会社名", product.companyName));

  appendRow(
    rows,
    createFormattedRow("category", "カテゴリ", resolveCategoryLabel(product)),
  );

  if (isAlcohol) {
    appendRow(
      rows,
      createRow(
        "material",
        "材料",
        getCategoryFieldValue(categoryFields, "material"),
      ),
    );

    appendRow(
      rows,
      createRow(
        "region",
        "生産地",
        getCategoryFieldValue(categoryFields, "region"),
      ),
    );

    appendRow(
      rows,
      createRow(
        "vintage",
        "ビンテージ",
        getCategoryFieldValue(categoryFields, "vintage"),
      ),
    );

    appendRow(
      rows,
      createFormattedRow(
        "alcoholContent",
        "アルコール度数",
        formatAlcoholContent(
          getCategoryFieldValue(categoryFields, "alcoholContent"),
        ),
      ),
    );
  }

  if (isApparel) {
    appendRow(
      rows,
      createRow("fit", "フィット", getCategoryFieldValue(categoryFields, "fit")),
    );

    appendRow(
      rows,
      createRow(
        "material",
        "素材",
        getCategoryFieldValue(categoryFields, "material"),
      ),
    );

    appendRow(
      rows,
      createRow(
        "weight",
        "重量",
        getCategoryFieldValue(categoryFields, "weight"),
      ),
    );
  }

  appendRow(
    rows,
    createRow("productIdTagType", "商品IDタグ", product.productIdTagType),
  );

  return {
    rows,
    qualityAssuranceItems: resolveQualityAssuranceItems(
      product.qualityAssurance,
    ),
  };
}

export type { ProductCategoryKind };