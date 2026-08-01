// frontend/console/shell/src/features/productBlueprintReview/application/productBlueprintReviewManagementService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";

import type {
  ListCompanyReviewAggregatesParams,
  ProductBlueprintReviewAggregate,
  ReviewStatus,
} from "../../../shared/types/productBlueprintReview";

import type {
  PageResult,
} from "../../../shared/types/common/common";

export async function FetchProductBlueprintReviewManagementRows(Params: {
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
}): Promise<PageResult<ProductBlueprintReviewAggregate>> {
  const Query: ListCompanyReviewAggregatesParams = {
    Status: Params.Status,
    Page: Params.Page,
    PerPage: Params.PerPage,
  };

  const Response =
    await productBlueprintReviewHTTP.ListCompanyReviewAggregates(
      Query,
    );

  return {
    items: Response.Items ?? [],
    totalCount: Response.TotalCount ?? 0,
    totalPages: Response.TotalPages ?? 0,
    page: Response.Page,
    perPage: Response.PerPage,
  };
}

export function FilterProductBlueprintReviewRows(Params: {
  AllRows: ProductBlueprintReviewAggregate[];
  BrandFilter: string[];
  AssigneeFilter: string[];
}): ProductBlueprintReviewAggregate[] {
  const {
    AllRows,
    BrandFilter,
    AssigneeFilter,
  } = Params;

  let FilteredRows = AllRows;

  if (BrandFilter.length > 0) {
    const BrandNames = new Set(BrandFilter);

    FilteredRows = FilteredRows.filter((Row) =>
      BrandNames.has(Row.BrandName),
    );
  }

  if (AssigneeFilter.length > 0) {
    const AssigneeNames = new Set(AssigneeFilter);

    FilteredRows = FilteredRows.filter((Row) =>
      AssigneeNames.has(Row.AssigneeName),
    );
  }

  return FilteredRows;
}