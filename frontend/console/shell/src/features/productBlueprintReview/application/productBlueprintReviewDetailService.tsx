// frontend/console/shell/src/features/productBlueprintReview/application/productBlueprintReviewDetailService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";
import { safeDateTimeLabelJa } from "../../../shared/util/dateJa";

import type {
  ListProductBlueprintReviewsParams,
  Review,
  ReviewStatus,
} from "../../../shared/types/productBlueprintReview";

export type ProductBlueprintReviewDetailRow = Review;

export type FetchProductBlueprintReviewDetailParams = {
  ProductBlueprintID: string;
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

export type FetchProductBlueprintReviewDetailResult = {
  Items: ProductBlueprintReviewDetailRow[];
  TotalPages: number;
};

export async function FetchProductBlueprintReviewDetailRows(
  Params: FetchProductBlueprintReviewDetailParams,
): Promise<FetchProductBlueprintReviewDetailResult> {
  const Query: ListProductBlueprintReviewsParams = {
    ProductBlueprintID: Params.ProductBlueprintID,
    Status: Params.Status,
    Page: Params.Page,
    PerPage: Params.PerPage,
  };

  const Response =
    await productBlueprintReviewHTTP.ListReviewsByProductBlueprintID(
      Query,
    );

  const Items: ProductBlueprintReviewDetailRow[] =
    Response.Items.map((ReviewItem) => ({
      ...ReviewItem,
      ReviewedAt: safeDateTimeLabelJa(
        ReviewItem.ReviewedAt,
        "",
      ),
    }));

  return {
    Items,
    TotalPages: Response.TotalPages ?? 0,
  };
}