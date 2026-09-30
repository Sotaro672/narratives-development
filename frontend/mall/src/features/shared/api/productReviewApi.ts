// frontend/mall/src/features/shared/api/productReviewApi.ts

import { requestJson } from "../../../lib/http";

import type { ProductBlueprintReview } from "../types/review";

export async function putProductReviewHelpfulVote(args: {
  productBlueprintId: string;
  reviewId: string;
  headers?: HeadersInit;
}): Promise<ProductBlueprintReview> {
  const productBlueprintId = args.productBlueprintId.trim();
  const reviewId = args.reviewId.trim();

  if (!productBlueprintId) {
    throw new Error("product review helpful vote: productBlueprintId is empty");
  }

  if (!reviewId) {
    throw new Error("product review helpful vote: reviewId is empty");
  }

  return requestJson<ProductBlueprintReview>(
    `/mall/me/catalog/product-blueprints/${encodeURIComponent(productBlueprintId)}/reviews/${encodeURIComponent(reviewId)}/helpful`,
    {
      method: "PUT",
      auth: "required",
      headers: args.headers,
      messages: {
        requestErrorMessage: "putProductReviewHelpfulVote failed",
        nonJsonErrorMessage: "putProductReviewHelpfulVote failed: response is not json",
        invalidJsonErrorMessage: "putProductReviewHelpfulVote failed: invalid json",
      },
    },
  );
}