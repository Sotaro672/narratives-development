// frontend/console/shell/src/features/productBlueprint/infrastructure/api/productBlueprintApi.ts

import { API_BASE } from "../../../../shared/http/apiBase";
import { getAuthHeadersOrThrow } from "../../../../shared/http/authHeaders";

import type {
  PageResult,
} from "../../../../shared/types/common/common";

import type {
  ProductBlueprintCategory,
  ProductBlueprintCategoryKind,
} from "../../domain/productBlueprintCategory";

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

export type ProductBlueprintCategoryTreeResponse = {
  items: ProductBlueprintCategory[];
};

type ProductBlueprintCategoryListResponse =
  PageResult<ProductBlueprintCategory>;

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function parseProductBlueprintCategoryItems(
  value: unknown,
  responseName: string,
): ProductBlueprintCategory[] {
  if (
    !isRecord(value) ||
    !Array.isArray(value.items)
  ) {
    throw new Error(
      `${responseName}のレスポンス形式が不正です。`,
    );
  }

  return value.items as ProductBlueprintCategory[];
}

async function getProductBlueprintCategories(
  url: string,
  responseName: string,
): Promise<ProductBlueprintCategory[]> {
  const headers =
    await getAuthHeadersOrThrow();

  const response =
    await fetch(
      url,
      {
        method: "GET",
        headers,
      },
    );

  if (!response.ok) {
    const detail =
      await response
        .text()
        .catch(
          () => "",
        );

    throw new Error(
      `${responseName}の取得に失敗しました（${response.status} ${response.statusText}）${
        detail
          ? `\n${detail}`
          : ""
      }`,
    );
  }

  const json: unknown =
    await response.json();

  return parseProductBlueprintCategoryItems(
    json,
    responseName,
  );
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
  const searchParams =
    new URLSearchParams();

  const queryParams: Array<
    [
      string,
      string | number | boolean | undefined,
    ]
  > = [
    [
      "kind",
      params?.kind,
    ],
    [
      "code",
      params?.code,
    ],
    [
      "parentId",
      params?.parentId,
    ],
    [
      "rootOnly",
      params?.rootOnly,
    ],
    [
      "search",
      params?.search,
    ],
    [
      "page",
      params?.page ?? 1,
    ],
    [
      "perPage",
      params?.perPage ?? 100,
    ],
    [
      "sort",
      params?.sort ?? "displayOrder",
    ],
    [
      "order",
      params?.order ?? "asc",
    ],
  ];

  for (
    const [
      key,
      value,
    ] of queryParams
  ) {
    if (
      value === undefined ||
      value === ""
    ) {
      continue;
    }

    searchParams.set(
      key,
      String(value),
    );
  }

  const query =
    searchParams.toString();

  const url =
    `${API_BASE}/console/product-blueprint-categories${
      query
        ? `?${query}`
        : ""
    }`;

  const items =
    await getProductBlueprintCategories(
      url,
      "商品カテゴリ一覧",
    );

  return items;
}

/**
 * GET /console/product-blueprint-categories/tree
 *
 * 商品カテゴリ選択UI向けに、
 * 親カテゴリと詳細カテゴリを含む一覧を
 * backendのdisplayOrder順で取得する。
 *
 * backend response:
 * {
 *   items: ProductBlueprintCategory[]
 * }
 */
export async function listProductBlueprintCategoryTreeApi():
  Promise<ProductBlueprintCategory[]> {
  const url =
    `${API_BASE}/console/product-blueprint-categories/tree`;

  const items =
    await getProductBlueprintCategories(
      url,
      "商品カテゴリツリー",
    );

  if (items.length === 0) {
    throw new Error(
      "商品カテゴリマスタが登録されていません。",
    );
  }

  return items;
}

// Response contracts used for compile-time consistency.
void (
  null as
    | ProductBlueprintCategoryListResponse
    | ProductBlueprintCategoryTreeResponse
    | null
);