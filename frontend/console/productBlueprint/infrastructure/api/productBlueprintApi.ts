// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintApi.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

import type {
  ProductBlueprintCategory,
  ProductBlueprintCategoryKind,
} from "../../domain/entity/productBlueprintCategory";

type ProductBlueprintCategoryListResponse = {
  items: ProductBlueprintCategory[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

type ProductBlueprintCategoryTreeResponse = {
  items: ProductBlueprintCategory[];
};

export type ListProductBlueprintCategoriesParams = {
  kind?: ProductBlueprintCategoryKind;
  code?: string;
  parentId?: string;
  rootOnly?: boolean;
  search?: string;
  page?: number;
  perPage?: number;
  sort?: string;
  order?: "asc" | "desc";
};

function sortProductBlueprintCategories(
  categories: ProductBlueprintCategory[],
): ProductBlueprintCategory[] {
  return [...categories].sort((a, b) => {
    const ao = Number(a.displayOrder ?? 0);
    const bo = Number(b.displayOrder ?? 0);

    if (ao !== bo) {
      return ao - bo;
    }

    return String(a.code ?? "").localeCompare(String(b.code ?? ""));
  });
}

function appendSearchParam(
  params: URLSearchParams,
  key: string,
  value: string | number | boolean | null | undefined,
): void {
  if (value === null || value === undefined || value === "") {
    return;
  }

  params.set(key, String(value));
}

function buildProductBlueprintCategoriesUrl(
  params?: ListProductBlueprintCategoriesParams,
): string {
  const searchParams = new URLSearchParams();

  appendSearchParam(searchParams, "kind", params?.kind);
  appendSearchParam(searchParams, "code", params?.code);
  appendSearchParam(searchParams, "parentId", params?.parentId);
  appendSearchParam(searchParams, "rootOnly", params?.rootOnly);
  appendSearchParam(searchParams, "search", params?.search);
  appendSearchParam(searchParams, "page", params?.page ?? 1);
  appendSearchParam(searchParams, "perPage", params?.perPage ?? 100);
  appendSearchParam(searchParams, "sort", params?.sort ?? "displayOrder");
  appendSearchParam(searchParams, "order", params?.order ?? "asc");

  const query = searchParams.toString();

  return `${API_BASE}/console/product-blueprint-categories${
    query ? `?${query}` : ""
  }`;
}

async function fetchJsonOrThrow<T>(url: string, errorMessage: string): Promise<T> {
  const headers = await getAuthHeadersOrThrow();

  const res = await fetch(url, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    const detail = await res.text().catch(() => "");
    throw new Error(
      `${errorMessage}（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  return (await res.json()) as T;
}

/**
 * GET /console/product-blueprint-categories
 *
 * backend response:
 * {
 *   items: ProductBlueprintCategory[],
 *   totalCount: number,
 *   totalPages: number,
 *   page: number,
 *   perPage: number
 * }
 */
export async function listProductBlueprintCategoriesApi(
  params?: ListProductBlueprintCategoriesParams,
): Promise<ProductBlueprintCategory[]> {
  const url = buildProductBlueprintCategoriesUrl(params);

  const json = await fetchJsonOrThrow<ProductBlueprintCategoryListResponse>(
    url,
    "商品カテゴリ一覧の取得に失敗しました",
  );

  return sortProductBlueprintCategories(json.items);
}

/**
 * GET /console/product-blueprint-categories?kind={kind}
 */
export async function listProductBlueprintCategoriesByKindApi(
  kind: ProductBlueprintCategoryKind,
): Promise<ProductBlueprintCategory[]> {
  return await listProductBlueprintCategoriesApi({
    kind,
    page: 1,
    perPage: 100,
    sort: "displayOrder",
    order: "asc",
  });
}

/**
 * code 指定でカテゴリを1件取得する。
 *
 * backend handler は GetByCode 専用 route ではなく、
 * List の query param code で絞り込む。
 */
export async function getProductBlueprintCategoryByCodeApi(
  code: string,
): Promise<ProductBlueprintCategory | null> {
  const trimmed = String(code ?? "").trim();

  if (!trimmed) {
    throw new Error("商品カテゴリコードが空です。");
  }

  const items = await listProductBlueprintCategoriesApi({
    code: trimmed,
    page: 1,
    perPage: 1,
  });

  return items[0] ?? null;
}

/**
 * GET /console/product-blueprint-categories/tree
 */
export async function listProductBlueprintCategoryTreeApi(): Promise<
  ProductBlueprintCategory[]
> {
  const url = `${API_BASE}/console/product-blueprint-categories/tree`;

  const json = await fetchJsonOrThrow<ProductBlueprintCategoryTreeResponse>(
    url,
    "商品カテゴリツリーの取得に失敗しました",
  );

  return sortProductBlueprintCategories(json.items);
}

/**
 * GET /console/product-blueprint-categories/{id}
 */
export async function getProductBlueprintCategoryByIdApi(
  id: string,
): Promise<ProductBlueprintCategory> {
  const trimmed = String(id ?? "").trim();

  if (!trimmed) {
    throw new Error("商品カテゴリIDが空です。");
  }

  const url = `${API_BASE}/console/product-blueprint-categories/${encodeURIComponent(
    trimmed,
  )}`;

  return await fetchJsonOrThrow<ProductBlueprintCategory>(
    url,
    "商品カテゴリの取得に失敗しました",
  );
}