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
  items?: ProductBlueprintCategory[];

  data?: {
    items?: ProductBlueprintCategory[];
  };
};

function sortProductBlueprintCategories(
  categories:
    ProductBlueprintCategory[],
): ProductBlueprintCategory[] {
  return [
    ...categories,
  ].sort(
    (
      a,
      b,
    ) => {
      const aDisplayOrder =
        Number(
          a.displayOrder ?? 0,
        );

      const bDisplayOrder =
        Number(
          b.displayOrder ?? 0,
        );

      if (
        aDisplayOrder !==
        bDisplayOrder
      ) {
        return (
          aDisplayOrder -
          bDisplayOrder
        );
      }

      return String(
        a.code ?? "",
      ).localeCompare(
        String(
          b.code ?? "",
        ),
      );
    },
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
  params?:
    ListProductBlueprintCategoriesParams,
): Promise<ProductBlueprintCategory[]> {
  const searchParams =
    new URLSearchParams();

  const queryParams: Array<
    [
      string,
      | string
      | number
      | boolean
      | null
      | undefined,
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
      params?.sort ??
        "displayOrder",
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
      value === null ||
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
      `商品カテゴリ一覧の取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  const json =
    await response.json() as
      PageResult<
        ProductBlueprintCategory
      > & {
        data?: PageResult<
          ProductBlueprintCategory
        >;
      };

  const items =
    json.items ??
    json.data?.items ??
    [];

  return sortProductBlueprintCategories(
    items,
  );
}

/**
 * GET /console/product-blueprint-categories/tree
 *
 * 商品カテゴリ選択UI向けに、
 * 親カテゴリと詳細カテゴリを含むツリー一覧を取得する。
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
      `商品カテゴリツリーの取得に失敗しました（${response.status} ${response.statusText}）\n${detail}`,
    );
  }

  const json =
    await response.json() as
      ProductBlueprintCategoryTreeResponse;

  const items =
    json.items ??
    json.data?.items ??
    [];

  return sortProductBlueprintCategories(
    items,
  );
}