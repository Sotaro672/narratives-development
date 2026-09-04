// frontend/amol/src/features/avatar-review/api/avatarReviewApi.ts

import { requestJson } from "../../../lib/http";

export type AvatarReviewEvaluation =
  | "good"
  | "disappointed";

export type AvatarReviewItem = {
  id: string;
  tradeId: string;
  orderId: string;
  orderItemIndex: number;
  reviewerAvatarId: string;
  revieweeAvatarId: string;
  evaluation: AvatarReviewEvaluation;
  comment: string;
  createdAt: string;
};

export type AvatarReviewPageResponse = {
  avatarId: string;
  goodCount: number;
  disappointedCount: number;
  total: number;
  page: number;
  perPage: number;
  hasNext: boolean;
  items: AvatarReviewItem[];
};

export async function fetchAvatarReviews(args: {
  avatarId: string;
  page?: number;
  perPage?: number;
}): Promise<AvatarReviewPageResponse> {
  const avatarId = args.avatarId.trim();

  if (!avatarId) {
    throw new Error("avatarId is empty");
  }

  return requestJson<AvatarReviewPageResponse>(
    `/mall/avatar-reviews/${encodeURIComponent(avatarId)}`,
    {
      method: "GET",
      query: {
        page: args.page ?? 1,
        perPage: args.perPage ?? 20,
      },
      unwrapData: true,
      messages: {
        requestErrorMessage:
          "fetchAvatarReviews failed",
        nonJsonErrorMessage:
          "fetchAvatarReviews failed: response is not json",
        invalidJsonErrorMessage:
          "fetchAvatarReviews failed: invalid json",
      },
    },
  );
}