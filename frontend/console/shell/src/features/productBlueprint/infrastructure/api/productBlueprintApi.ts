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
      string | number | boolean | null | undefined,
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

  const res =
    await fetch(
      url,
      {
        method: "GET",
        headers,
      },
    );

  if (!res.ok) {
    const detail =
      await res
        .text()
        .catch(
          () => "",
        );

    throw new Error(
      `商品カテゴリ一覧の取得に失敗しました（${res.status} ${res.statusText}）\n${detail}`,
    );
  }

  const json =
    await res.json() as PageResult<
      ProductBlueprintCategory
    >;

  return [
    ...(json.items ?? []),
  ].sort(
    (
      a,
      b,
    ) => {
      const ao =
        Number(
          a.displayOrder ?? 0,
        );

      const bo =
        Number(
          b.displayOrder ?? 0,
        );

      if (ao !== bo) {
        return ao - bo;
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