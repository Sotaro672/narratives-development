// frontend/console/shell/src/features/productBlueprintReview/application/productBlueprintReviewDetailService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

import type {
  ListProductBlueprintReviewsParams,
  Review,
  ReviewStatus,
} from "../../../shared/types/productBlueprintReview";
import type {
  ReportReason,
  ReportResponse,
} from "../../../shared/types/report";
import type {
  PageResult,
} from "../../../shared/types/common/common";

export type FetchProductBlueprintReviewDetailParams = {
  ProductBlueprintID: string;
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

export type FetchProductBlueprintReviewDetailResult =
  PageResult<Review>;

export async function FetchProductBlueprintReviewDetailRows(
  Params: FetchProductBlueprintReviewDetailParams,
): Promise<FetchProductBlueprintReviewDetailResult> {
  const ProductBlueprintID = Params.ProductBlueprintID.trim();

  if (!ProductBlueprintID) {
    throw new Error("ProductBlueprintID is required");
  }

  const Query: ListProductBlueprintReviewsParams = {
    ProductBlueprintID,
    Status: Params.Status,
    Page: Params.Page,
    PerPage: Params.PerPage,
  };

  const Response =
    await productBlueprintReviewHTTP.ListReviewsByProductBlueprintID(
      Query,
    );

  const items: Review[] = Response.Items.map((ReviewItem) => ({
    ...ReviewItem,
    ReviewedAt: safeDateTimeLabelJa(
      ReviewItem.ReviewedAt,
      "",
    ),
  }));

  return {
    items,
    totalCount: Response.TotalCount,
    totalPages: Response.TotalPages,
    page: Response.Page,
    perPage: Response.PerPage,
  };
}

export async function ReportProductBlueprintReview(
  ProductBlueprintID: string,
  ReviewID: string,
  Reason: ReportReason,
  Detail?: string,
): Promise<ReportResponse> {
  const NormalizedProductBlueprintID =
    ProductBlueprintID.trim();
  const NormalizedReviewID =
    ReviewID.trim();
  const NormalizedDetail =
    Detail?.trim() ?? "";

  if (!NormalizedProductBlueprintID) {
    throw new Error("ProductBlueprintID is required");
  }

  if (!NormalizedReviewID) {
    throw new Error("ReviewID is required");
  }

  return productBlueprintReviewHTTP.ReportProductBlueprintReview({
    productBlueprintId: NormalizedProductBlueprintID,
    reviewId: NormalizedReviewID,
    reason: Reason,
    ...(NormalizedDetail
      ? { detail: NormalizedDetail }
      : {}),
  });
}