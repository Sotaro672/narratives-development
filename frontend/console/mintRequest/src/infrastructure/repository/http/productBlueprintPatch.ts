// frontend/console/mintRequest/src/infrastructure/repository/http/productBlueprintPatch.ts

import { API_BASE } from "../../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../../shell/src/shared/http/authHeaders";

import type { ProductBlueprintPatchDTO } from "../../dto/mintRequestLocal.dto";

import type {
  ProductBlueprintCategoryKind,
  ProductBlueprintCategorySnapshot,
  CategoryFieldValues,
  CategoryFieldValue,
} from "../../../../../productBlueprint/src/domain/productBlueprintCategory";

import { isValidProductBlueprintCategoryKind } from "../../../../../productBlueprint/src/domain/productBlueprintCategory";

type ProductBlueprintCategoryRaw = {
  ID?: unknown;
  id?: unknown;

  Code?: unknown;
  code?: unknown;

  NameJa?: unknown;
  nameJa?: unknown;

  NameEn?: unknown;
  nameEn?: unknown;

  ParentID?: unknown;
  ParentId?: unknown;
  parentId?: unknown;

  Kind?: unknown;
  kind?: unknown;

  Path?: unknown;
  path?: unknown;

  DisplayOrder?: unknown;
  displayOrder?: unknown;
};

type ProductBlueprintModelRefRaw = {
  ModelID?: unknown;
  modelId?: unknown;

  DisplayOrder?: unknown;
  displayOrder?: unknown;
};

type ProductBlueprintPatchRaw = {
  ID?: unknown;
  id?: unknown;

  ProductName?: unknown;
  productName?: unknown;

  Description?: unknown;
  description?: unknown;

  CompanyID?: unknown;
  companyId?: unknown;

  BrandID?: unknown;
  brandId?: unknown;

  BrandName?: unknown;
  brandName?: unknown;

  ProductBlueprintCategory?: unknown;
  productBlueprintCategory?: unknown;

  CategoryFields?: unknown;
  categoryFields?: unknown;

  ProductIdTag?: unknown;
  productIdTag?: unknown;

  AssigneeID?: unknown;
  assigneeId?: unknown;

  ModelRefs?: unknown;
  modelRefs?: unknown;

  Printed?: unknown;
  printed?: unknown;

  CreatedBy?: unknown;
  createdBy?: unknown;

  CreatedAt?: unknown;
  createdAt?: unknown;

  UpdatedBy?: unknown;
  updatedBy?: unknown;

  UpdatedAt?: unknown;
  updatedAt?: unknown;
};

const toText = (value: unknown): string => {
  return typeof value === "string" ? value.trim() : "";
};

const toNullableText = (value: unknown): string | null => {
  const text = toText(value);
  return text || null;
};

const toNumberOrUndefined = (value: unknown): number | undefined => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return undefined;

    const parsed = Number(trimmed);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  return undefined;
};

const toStringArray = (value: unknown): string[] => {
  if (!Array.isArray(value)) return [];

  return value
    .map((item) => toText(item))
    .filter((item) => item.length > 0);
};

const toCategoryKind = (
  value: unknown,
): ProductBlueprintCategoryKind | "other" => {
  const text = toText(value);

  if (isValidProductBlueprintCategoryKind(text)) {
    return text;
  }

  return "other";
};

const isCategoryFieldValue = (
  value: unknown,
): value is CategoryFieldValue => {
  if (
    value === null ||
    typeof value === "string" ||
    typeof value === "number" ||
    typeof value === "boolean"
  ) {
    return true;
  }

  if (Array.isArray(value)) {
    return value.every(
      (item) =>
        item === null ||
        typeof item === "string" ||
        typeof item === "number" ||
        typeof item === "boolean",
    );
  }

  if (typeof value === "object") {
    return Object.values(value as Record<string, unknown>).every(
      (item) =>
        item === null ||
        typeof item === "string" ||
        typeof item === "number" ||
        typeof item === "boolean",
    );
  }

  return false;
};

const toCategoryFieldValues = (
  value: unknown,
): CategoryFieldValues | null => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const entries = Object.entries(value as Record<string, unknown>).filter(
    ([key, item]) => key.trim().length > 0 && isCategoryFieldValue(item),
  );

  return Object.fromEntries(entries) as CategoryFieldValues;
};

const toProductBlueprintCategorySnapshot = (
  value: unknown,
): ProductBlueprintCategorySnapshot | null => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const raw = value as ProductBlueprintCategoryRaw;

  const id = toText(raw.id) || toText(raw.ID);
  const code = toText(raw.code) || toText(raw.Code);
  const nameJa = toText(raw.nameJa) || toText(raw.NameJa);
  const nameEn = toText(raw.nameEn) || toText(raw.NameEn);
  const kind = toCategoryKind(raw.kind ?? raw.Kind);
  const path = toStringArray(raw.path ?? raw.Path);

  if (!id && !code) {
    return null;
  }

  const parentId =
    toNullableText(raw.parentId) ??
    toNullableText(raw.ParentId) ??
    toNullableText(raw.ParentID);

  const displayOrder =
    toNumberOrUndefined(raw.displayOrder) ??
    toNumberOrUndefined(raw.DisplayOrder);

  return {
    id: id || code,
    code: code || id,
    nameJa,
    nameEn,
    parentId,
    kind,
    path,
    displayOrder,
  };
};

const toModelRefs = (
  value: unknown,
): ProductBlueprintPatchDTO["modelRefs"] => {
  if (!Array.isArray(value)) {
    return null;
  }

  return value
    .map(
      (
        item,
      ): ProductBlueprintPatchDTO["modelRefs"] extends
        | Array<infer T>
        | null
        | undefined
        ? T | null
        : never => {
        if (!item || typeof item !== "object" || Array.isArray(item)) {
          return null as never;
        }

        const raw = item as ProductBlueprintModelRefRaw;

        const modelId = toText(raw.modelId) || toText(raw.ModelID);
        if (!modelId) {
          return null as never;
        }

        return {
          modelId,
          displayOrder:
            toNumberOrUndefined(raw.displayOrder) ??
            toNumberOrUndefined(raw.DisplayOrder) ??
            0,
        } as never;
      },
    )
    .filter(
      (
        item,
      ): item is NonNullable<ProductBlueprintPatchDTO["modelRefs"]>[number] =>
        item !== null,
    );
};

const toProductIdTag = (
  value: unknown,
): ProductBlueprintPatchDTO["productIdTag"] => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const raw = value as { type?: unknown; Type?: unknown };

  return {
    type: toNullableText(raw.type),
    Type: toNullableText(raw.Type),
  };
};

const toProductBlueprintPatchDTO = (
  value: unknown,
): ProductBlueprintPatchDTO | null => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const raw = value as ProductBlueprintPatchRaw;

  return {
    productName:
      toNullableText(raw.productName) ??
      toNullableText(raw.ProductName),

    description:
      toNullableText(raw.description) ??
      toNullableText(raw.Description),

    brandId:
      toNullableText(raw.brandId) ??
      toNullableText(raw.BrandID),

    brandName:
      toNullableText(raw.brandName) ??
      toNullableText(raw.BrandName),

    companyId:
      toNullableText(raw.companyId) ??
      toNullableText(raw.CompanyID),

    productBlueprintCategory: toProductBlueprintCategorySnapshot(
      raw.productBlueprintCategory ?? raw.ProductBlueprintCategory,
    ),

    categoryFields: toCategoryFieldValues(
      raw.categoryFields ?? raw.CategoryFields,
    ),

    productIdTag: toProductIdTag(
      raw.productIdTag ?? raw.ProductIdTag,
    ),

    assigneeId:
      toNullableText(raw.assigneeId) ??
      toNullableText(raw.AssigneeID),

    modelRefs: toModelRefs(raw.modelRefs ?? raw.ModelRefs),
  };
};

export async function fetchProductBlueprintPatchHTTP(
  productBlueprintId: string,
): Promise<ProductBlueprintPatchDTO | null> {
  const pbid = String(productBlueprintId ?? "").trim();
  if (!pbid) throw new Error("productBlueprintId が空です");

  const authHeaders = await getAuthHeadersOrThrow();

  const url = `${API_BASE}/mint/product_blueprints/${encodeURIComponent(
    pbid,
  )}`;

  const res = await fetch(url, { method: "GET", headers: authHeaders });

  if (res.status === 404) return null;

  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(
      `Failed to fetch productBlueprint: ${res.status} ${res.statusText}${
        body ? ` body=${body.slice(0, 400)}` : ""
      }`,
    );
  }

  const json = (await res.json()) as unknown;
  return toProductBlueprintPatchDTO(json);
}