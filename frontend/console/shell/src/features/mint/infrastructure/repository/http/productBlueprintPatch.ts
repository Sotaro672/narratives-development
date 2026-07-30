// frontend/console/shell/src/features/mint/infrastructure/repository/http/productBlueprintPatch.ts

import {
  API_BASE,
} from "../../../../../shared/http/apiBase";

import {
  getAuthHeadersOrThrow,
} from "../../../../../shared/http/authHeaders";

import type {
  ProductIDTag,
} from "../../../../../shared/types/productBlueprint";

import {
  isValidProductIDTagType,
} from "../../../../../shared/types/productBlueprint";

import type {
  ProductBlueprintModelRefDTO,
  ProductBlueprintPatchDTO,
} from "../../dto/mintRequestLocal.dto";

import type {
  CategoryFieldPrimitiveValue,
  CategoryFieldValue,
  CategoryFieldValues,
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
} from "../../../../productBlueprint/domain/productBlueprintCategory";

import {
  isValidProductBlueprintCategoryKind,
} from "../../../../productBlueprint/domain/productBlueprintCategory";

/**
 * GET /mint/product_blueprints/{id}が返す
 * ProductBlueprintCategorySnapshotのRaw形式。
 *
 * BackendのGo構造体にはJSONタグがないため、
 * Goの公開フィールド名をそのまま受け取る。
 */
type ProductBlueprintCategoryRaw = {
  ID?: unknown;
  Code?: unknown;
  NameJa?: unknown;
  NameEn?: unknown;
  Kind?: unknown;
  Path?: unknown;
};

/**
 * BackendのproductBlueprint.ModelRef。
 */
type ProductBlueprintModelRefRaw = {
  ModelID?: unknown;
  DisplayOrder?: unknown;
};

/**
 * BackendのproductBlueprint.ProductIDTag。
 */
type ProductIDTagRaw = {
  Type?: unknown;
};

/**
 * GET /mint/product_blueprints/{id}のRaw形式。
 *
 * BackendのProductBlueprintはJSONタグを持たないため、
 * Goの公開フィールド名を正とする。
 *
 * BrandNameだけはMintProductBlueprintDTO側で
 * json:"brandName"が指定されている。
 */
type ProductBlueprintPatchRaw = {
  ProductName?: unknown;
  Description?: unknown;

  CompanyID?: unknown;
  BrandID?: unknown;
  brandName?: unknown;

  ProductBlueprintCategory?: unknown;
  CategoryFields?: unknown;

  ProductIdTag?: unknown;
  AssigneeID?: unknown;
  ModelRefs?: unknown;
};

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function toText(
  value: unknown,
): string {
  return typeof value === "string"
    ? value.trim()
    : "";
}

function toNullableText(
  value: unknown,
): string | null {
  const text =
    toText(value);

  return text || null;
}

function toNumberOrUndefined(
  value: unknown,
): number | undefined {
  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return value;
  }

  return undefined;
}

function toStringArray(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map(toText)
    .filter(
      (item) =>
        item.length > 0,
    );
}

function toCategoryKind(
  value: unknown,
): ProductBlueprintCategoryKind {
  const text =
    toText(value);

  if (
    isValidProductBlueprintCategoryKind(
      text,
    )
  ) {
    return text;
  }

  return "other";
}

function isCategoryFieldPrimitiveValue(
  value: unknown,
): value is CategoryFieldPrimitiveValue {
  return (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  );
}

function isCategoryFieldValue(
  value: unknown,
): value is CategoryFieldValue {
  if (
    isCategoryFieldPrimitiveValue(
      value,
    )
  ) {
    return true;
  }

  if (Array.isArray(value)) {
    return value.every(
      isCategoryFieldPrimitiveValue,
    );
  }

  if (isRecord(value)) {
    return Object.values(
      value,
    ).every(
      isCategoryFieldPrimitiveValue,
    );
  }

  return false;
}

function toCategoryFieldValues(
  value: unknown,
): CategoryFieldValues | null {
  if (!isRecord(value)) {
    return null;
  }

  const entries =
    Object.entries(value).filter(
      ([key, item]) =>
        key.trim().length > 0 &&
        isCategoryFieldValue(item),
    );

  return Object.fromEntries(
    entries,
  ) as CategoryFieldValues;
}

function toProductBlueprintCategorySnapshot(
  value: unknown,
): ProductBlueprintCategorySnapshot | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw =
    value as ProductBlueprintCategoryRaw;

  const id =
    toText(raw.ID);

  const code =
    toText(raw.Code);

  if (
    !id ||
    !code
  ) {
    return null;
  }

  return {
    id,
    code,

    nameJa:
      toText(raw.NameJa),

    nameEn:
      toText(raw.NameEn),

    kind:
      toCategoryKind(
        raw.Kind,
      ),

    path:
      toStringArray(
        raw.Path,
      ),
  };
}

function toModelRefs(
  value: unknown,
): ProductBlueprintModelRefDTO[] | null {
  if (!Array.isArray(value)) {
    return null;
  }

  const modelRefs:
    ProductBlueprintModelRefDTO[] = [];

  for (const item of value) {
    if (!isRecord(item)) {
      continue;
    }

    const raw =
      item as ProductBlueprintModelRefRaw;

    const modelId =
      toText(
        raw.ModelID,
      );

    if (!modelId) {
      continue;
    }

    modelRefs.push({
      modelId,

      displayOrder:
        toNumberOrUndefined(
          raw.DisplayOrder,
        ) ??
        0,
    });
  }

  return modelRefs;
}

function toProductIdTag(
  value: unknown,
): ProductIDTag | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw =
    value as ProductIDTagRaw;

  const type =
    toText(raw.Type);

  if (
    !isValidProductIDTagType(
      type,
    )
  ) {
    return null;
  }

  return {
    type,
  };
}

function toProductBlueprintPatchDTO(
  value: unknown,
): ProductBlueprintPatchDTO | null {
  if (!isRecord(value)) {
    return null;
  }

  const raw =
    value as ProductBlueprintPatchRaw;

  return {
    productName:
      toNullableText(
        raw.ProductName,
      ),

    description:
      toNullableText(
        raw.Description,
      ),

    brandId:
      toNullableText(
        raw.BrandID,
      ),

    brandName:
      toNullableText(
        raw.brandName,
      ),

    companyId:
      toNullableText(
        raw.CompanyID,
      ),

    productBlueprintCategory:
      toProductBlueprintCategorySnapshot(
        raw.ProductBlueprintCategory,
      ),

    categoryFields:
      toCategoryFieldValues(
        raw.CategoryFields,
      ),

    productIdTag:
      toProductIdTag(
        raw.ProductIdTag,
      ),

    assigneeId:
      toNullableText(
        raw.AssigneeID,
      ),

    modelRefs:
      toModelRefs(
        raw.ModelRefs,
      ),
  };
}

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const normalizedProductBlueprintId =
    String(
      productBlueprintId ?? "",
    ).trim();

  if (!normalizedProductBlueprintId) {
    throw new Error(
      "productBlueprintId が空です",
    );
  }

  const authHeaders =
    await getAuthHeadersOrThrow();

  const url =
    `${API_BASE}/mint/product_blueprints/` +
    encodeURIComponent(
      normalizedProductBlueprintId,
    );

  const response =
    await fetch(
      url,
      {
        method: "GET",
        headers: authHeaders,
      },
    );

  if (response.status === 404) {
    return null;
  }

  if (!response.ok) {
    const body =
      await response
        .text()
        .catch(() => "");

    throw new Error(
      `Failed to fetch productBlueprint: ` +
        `${response.status} ` +
        `${response.statusText}` +
        (
          body
            ? ` body=${body.slice(0, 400)}`
            : ""
        ),
    );
  }

  const json =
    await response.json() as unknown;

  return toProductBlueprintPatchDTO(
    json,
  );
}