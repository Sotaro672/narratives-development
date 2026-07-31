// frontend/console/shell/src/features/productBlueprintReview/application/productBlueprintReviewManagementService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";

import type {
  ListCompanyReviewAggregatesParams,
  ProductBlueprintReviewAggregate,
  ReviewStatus,
} from "../../../shared/types/productBluleprintReview";

export type UiRow = ProductBlueprintReviewAggregate;

export async function FetchProductBlueprintReviewManagementRows(Params: {
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
}): Promise<UiRow[]> {
  const Query: ListCompanyReviewAggregatesParams = {
    Status: Params.Status,
    Page: Params.Page,
    PerPage: Params.PerPage,
  };

  const Response =
    await productBlueprintReviewHTTP.ListCompanyReviewAggregates(Query);

  return Response.Items ?? [];
}

export function FilterProductBlueprintReviewRows(Params: {
  AllRows: UiRow[];
  BrandFilter: string[];
  AssigneeFilter: string[];
}): UiRow[] {
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