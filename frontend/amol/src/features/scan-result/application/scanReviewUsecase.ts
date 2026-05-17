// frontend/amol/src/features/scan-result/application/scanReviewUsecase.ts
import type { CatalogReviewPage } from "../types";

export type ScanReviewUsecaseDeps = {
  fetchReviewsByProductBlueprintId: (input: {
    productBlueprintId: string;
    page: number;
    perPage: number;
  }) => Promise<CatalogReviewPage>;

  createProductBlueprintReview: (input: {
    productBlueprintId: string;
    body: string;
    rating: number;
    title: string;
    headers?: HeadersInit;
  }) => Promise<unknown>;

  getAuthHeadersOrUndefined: () => Promise<HeadersInit | undefined>;
};

export type SubmitScanReviewInput = {
  productBlueprintId: string;
  body: string;
  rating: number;
};

export function validateScanReviewInput(
  input: SubmitScanReviewInput,
): string | null {
  if (!input.body.trim()) {
    return "本文を入力してください";
  }

  if (!input.productBlueprintId.trim()) {
    return "productBlueprintId が取得できませんでした";
  }

  return null;
}

export function toScanReviewErrorMessage(error: unknown): string {
  const message = error instanceof Error ? error.message : String(error);

  if (message.includes("verified purchase required") || message.includes("403")) {
    return "購入済み（Verified）の方のみ投稿できます";
  }

  return message;
}

export async function submitScanReview(
  deps: ScanReviewUsecaseDeps,
  input: SubmitScanReviewInput,
): Promise<void> {
  const validationError = validateScanReviewInput(input);

  if (validationError) {
    throw new Error(validationError);
  }

  const headers = await deps.getAuthHeadersOrUndefined();

  await deps.createProductBlueprintReview({
    productBlueprintId: input.productBlueprintId.trim(),
    body: input.body.trim(),
    rating: input.rating,
    title: "Review",
    headers,
  });
}

export async function loadScanReviews(
  deps: ScanReviewUsecaseDeps,
  input: {
    productBlueprintId: string;
    page: number;
    perPage: number;
  },
): Promise<CatalogReviewPage> {
  const productBlueprintId = input.productBlueprintId.trim();

  if (!productBlueprintId) {
    throw new Error("productBlueprintId is empty");
  }

  return deps.fetchReviewsByProductBlueprintId({
    productBlueprintId,
    page: input.page,
    perPage: input.perPage,
  });
}