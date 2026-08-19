// frontend/console/shell/src/features/productBlueprint/infrastructure/api/productBlueprintApi.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeaders } from "../../../../shared/http/authHeaders";
import type { PageResult } from "../../../../shared/types/common/common";
import type { ProductBlueprintCategoryPath } from "../../domain/productBlueprintCategory";

export type ListProductBlueprintCategoriesParams = {
  paths?: ProductBlueprintCategoryPath[];
  page?: number;
  perPage?: number;
  sort?: "productBlueprintCategoryPath";
  order?: "asc" | "desc";
};

type ProductBlueprintCategoryApiItem = {
  productBlueprintCategoryPath: ProductBlueprintCategoryPath;
};

export type ProductBlueprintCategoryTreeResponse = {
  items: ProductBlueprintCategoryApiItem[];
};

type ProductBlueprintCategoryListResponse = PageResult<ProductBlueprintCategoryApiItem>;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isProductBlueprintCategoryPath(value: unknown): value is ProductBlueprintCategoryPath {
  return (
    Array.isArray(value) &&
    value.length > 0 &&
    value.every((segment) => typeof segment === "string" && segment !== "")
  );
}

function parseProductBlueprintCategoryPaths(
  value: unknown,
  responseName: string,
): ProductBlueprintCategoryPath[] {
  if (!isRecord(value) || !Array.isArray(value.items)) {
    throw new Error(`${responseName}のレスポンス形式が不正です。`);
  }

  return value.items.map((item) => {
    if (!isRecord(item) || !isProductBlueprintCategoryPath(item.productBlueprintCategoryPath)) {
      throw new Error(`${responseName}の商品カテゴリパスが不正です。`);
    }

    return [...item.productBlueprintCategoryPath];
  });
}

async function getProductBlueprintCategoryPaths(
  url: string,
  responseName: string,
): Promise<ProductBlueprintCategoryPath[]> {
  const headers = await getAuthHeaders();
  const response = await fetch(url, {
    method: "GET",
    headers,
  });

  if (!response.ok) {
    const detail = await response.text().catch(() => "");
    throw new Error(
      `${responseName}の取得に失敗しました（${response.status} ${response.statusText}）${
        detail ? `\n${detail}` : ""
      }`,
    );
  }

  const json: unknown = await response.json();
  return parseProductBlueprintCategoryPaths(json, responseName);
}

/**
 * GET /console/product-blueprint-categories
 *
 * backend response:
 * {
 *   items: [
 *     {
 *       productBlueprintCategoryPath: string[]
 *     }
 *   ],
 *   totalCount: number,
 *   totalPages: number,
 *   page: number,
 *   perPage: number
 * }
 */
export async function listProductBlueprintCategoriesApi(
  params?: ListProductBlueprintCategoriesParams,
): Promise<ProductBlueprintCategoryPath[]> {
  const searchParams = new URLSearchParams();

  if (params?.paths && params.paths.length > 0) {
    searchParams.set(
      "paths",
      params.paths.map((path) => path.join("/")).join(","),
    );
  }

  searchParams.set("page", String(params?.page ?? 1));
  searchParams.set("perPage", String(params?.perPage ?? 100));
  searchParams.set("sort", params?.sort ?? "productBlueprintCategoryPath");
  searchParams.set("order", params?.order ?? "asc");

  const query = searchParams.toString();
  const url = `${API_BASE}/console/product-blueprint-categories${query ? `?${query}` : ""}`;

  return getProductBlueprintCategoryPaths(url, "商品カテゴリ一覧");
}

/**
 * GET /console/product-blueprint-categories/tree
 *
 * backend response:
 * {
 *   items: [
 *     {
 *       productBlueprintCategoryPath: string[]
 *     }
 *   ]
 * }
 */
export async function listProductBlueprintCategoryTreeApi(): Promise<ProductBlueprintCategoryPath[]> {
  const url = `${API_BASE}/console/product-blueprint-categories/tree`;
  const paths = await getProductBlueprintCategoryPaths(url, "商品カテゴリツリー");

  if (paths.length === 0) {
    throw new Error("商品カテゴリマスタが登録されていません。");
  }

  return paths;
}

// Response contracts used for compile-time consistency.
void (
  null as
    | ProductBlueprintCategoryListResponse
    | ProductBlueprintCategoryTreeResponse
    | null
);