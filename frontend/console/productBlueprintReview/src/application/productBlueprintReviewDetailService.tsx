// frontend/console/productBlueprintReview/src/application/productBlueprintReviewDetailService.tsx

import { productBlueprintReviewHTTP } from "../infrastructure/productBlueprintReviewHTTP";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";

import type {
  ListProductBlueprintReviewsParams,
  ListProductBlueprintReviewsResponse,
  Review,
  ReviewStatus,
} from "../domain/entity";

/**
 * Detail 画面で扱う ViewModel（PascalCase）
 * - Backendの Review をそのまま返す（最終的に画面へ list して渡すため）
 */
export type ProductBlueprintReviewDetailRow = Review;

export type FetchProductBlueprintReviewDetailParams = {
  ProductBlueprintID: string; // route param をそのまま渡す想定（aggregate docId == pbID）
  Status?: ReviewStatus;
  Page?: number;
  PerPage?: number;
};

/**
 * ✅ Detail 用: 指定 ProductBlueprintID の reviews を取得して返す
 * backend: GET /product-blueprint-reviews?ProductBlueprintID=...&Status=...&Page=...&PerPage=...
 */
export async function FetchProductBlueprintReviewDetailRows(
  Params: FetchProductBlueprintReviewDetailParams,
): Promise<{
  ProductBlueprintID: string;
  Status: ReviewStatus;
  Page: number;
  PerPage: number;
  Items: ProductBlueprintReviewDetailRow[];
  TotalCount: number;
  TotalPages: number;
}> {
  const { ProductBlueprintID, Status, Page, PerPage } = Params;

  if (!String(ProductBlueprintID || "").trim()) {
    return {
      ProductBlueprintID: "",
      Status: (Status ?? "PUBLISHED") as ReviewStatus,
      Page: Page ?? 1,
      PerPage: PerPage ?? 20,
      Items: [],
      TotalCount: 0,
      TotalPages: 0,
    };
  }

  const Q: ListProductBlueprintReviewsParams = {
    ProductBlueprintID: String(ProductBlueprintID),
    Status,
    Page,
    PerPage,
  };

  const Res: ListProductBlueprintReviewsResponse =
    await productBlueprintReviewHTTP.ListReviewsByProductBlueprintID(Q);

  const Items: ProductBlueprintReviewDetailRow[] = (Res.Items ?? []).map((r) => ({
    ...r,
    ReviewedAt: safeDateTimeLabelJa(r.ReviewedAt, ""),
  })) as ProductBlueprintReviewDetailRow[];

  return {
    ProductBlueprintID: Res.ProductBlueprintID,
    Status: Res.Status,
    Page: Res.Page,
    PerPage: Res.PerPage,
    Items,
    TotalCount: Res.TotalCount ?? 0,
    TotalPages: Res.TotalPages ?? 0,
  };
}