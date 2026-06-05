// frontend/console/productBlueprint/src/infrastructure/api/productBlueprintCategoryApi.ts

import { API_BASE } from "../../../../shell/src/shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shell/src/shared/http/authHeaders";

import type {
  ProductBlueprintCategory,
  ProductBlueprintCategoryKind,
} from "../../domain/entity/productBlueprintCategory";

type ProductBlueprintCategoryRaw = ProductBlueprintCategory & {
  id?: string;
  code?: string;
  nameJa?: string;
  nameEn?: string;
  kind?: ProductBlueprintCategoryKind;
  path?: string[];

  ID?: string;
  Code?: string;
  NameJa?: string;
  NameEn?: string;
  Kind?: ProductBlueprintCategoryKind;
  Path?: string[];

  parentId?: string | null;
  ParentID?: string | null;
  ParentId?: string | null;

  displayOrder?: number | null;
  DisplayOrder?: number | null;
};

type ProductBlueprintCategoryListResponse = {
  items: ProductBlueprintCategoryRaw[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

type ProductBlueprintCategoryTreeResponse = {
  items: ProductBlueprintCategoryRaw[];
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

function normalizeProductBlueprintCategory(
  raw: ProductBlueprintCategoryRaw,
): ProductBlueprintCategory {
  const id = String(raw.id ?? raw.ID ?? "");
  const code = String(raw.code ?? raw.Code ?? id);
  const nameJa = String(raw.nameJa ?? raw.NameJa ?? "");
  const nameEn = String(raw.nameEn ?? raw.NameEn ?? "");
  const kind = String(raw.kind ?? raw.Kind ?? "") as ProductBlueprintCategoryKind;

  const rawPath = raw.path ?? raw.Path;
  const path = Array.isArray(rawPath)
    ? rawPath.map((value) => String(value)).filter(Boolean)
    : code
      ? code.split(".").filter(Boolean)
      : [];

  const parentId = raw.parentId ?? raw.ParentID ?? raw.ParentId ?? null;
  const displayOrder = raw.displayOrder ?? raw.DisplayOrder ?? 0;

  return {
    ...raw,
    id,
    code,
    nameJa,
    nameEn,
    kind,
    path,
    parentId,
    displayOrder,
  };
}

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

export async function listProductBlueprintCategoriesApi(
  params?: ListProductBlueprintCategoriesParams,
): Promise<ProductBlueprintCategory[]> {
  const url = buildProductBlueprintCategoriesUrl(params);

  const json = await fetchJsonOrThrow<ProductBlueprintCategoryListResponse>(
    url,
    "商品カテゴリ一覧の取得に失敗しました",
  );

  const items = (json.items ?? []).map(normalizeProductBlueprintCategory);

  return sortProductBlueprintCategories(items);
}

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

export async function listProductBlueprintCategoryTreeApi(): Promise<
  ProductBlueprintCategory[]
> {
  const url = `${API_BASE}/console/product-blueprint-categories/tree`;

  const json = await fetchJsonOrThrow<ProductBlueprintCategoryTreeResponse>(
    url,
    "商品カテゴリツリーの取得に失敗しました",
  );

  const items = (json.items ?? []).map(normalizeProductBlueprintCategory);

  return sortProductBlueprintCategories(items);
}

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

  const json = await fetchJsonOrThrow<ProductBlueprintCategoryRaw>(
    url,
    "商品カテゴリの取得に失敗しました",
  );

  return normalizeProductBlueprintCategory(json);
}