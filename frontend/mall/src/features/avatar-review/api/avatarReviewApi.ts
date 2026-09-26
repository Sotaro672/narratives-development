// frontend/mall/src/features/avatar-review/api/avatarReviewApi.ts

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

export type AvatarReviewStatusResponse = {
  eligible: boolean;
  reviewed: boolean;
  tradeId: string;
  orderId: string;
  orderItemIndex: number;
  revieweeAvatarId: string;
};

export type CreateAvatarReviewInput = {
  orderId: string;
  orderItemIndex: number;
  evaluation: AvatarReviewEvaluation;
  comment: string;
};

function normalizeOrderId(orderId: string): string {
  const normalized = orderId.trim();

  if (!normalized) {
    throw new Error("orderId is empty");
  }

  return normalized;
}

function normalizeOrderItemIndex(orderItemIndex: number): number {
  if (
    !Number.isInteger(orderItemIndex) ||
    orderItemIndex < 0
  ) {
    throw new Error("orderItemIndex is invalid");
  }

  return orderItemIndex;
}

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

export async function fetchAvatarReviewStatus(args: {
  orderId: string;
  orderItemIndex: number;
}): Promise<AvatarReviewStatusResponse> {
  const orderId = normalizeOrderId(args.orderId);
  const orderItemIndex = normalizeOrderItemIndex(
    args.orderItemIndex,
  );

  return requestJson<AvatarReviewStatusResponse>(
    `/mall/me/avatar-reviews/order-items/${encodeURIComponent(orderId)}/${orderItemIndex}`,
    {
      method: "GET",
      auth: "required",
      unwrapData: true,
      messages: {
        requestErrorMessage:
          "fetchAvatarReviewStatus failed",
        nonJsonErrorMessage:
          "fetchAvatarReviewStatus failed: response is not json",
        invalidJsonErrorMessage:
          "fetchAvatarReviewStatus failed: invalid json",
      },
    },
  );
}

export async function createAvatarReview(
  args: CreateAvatarReviewInput,
): Promise<AvatarReviewItem> {
  const orderId = normalizeOrderId(args.orderId);
  const orderItemIndex = normalizeOrderItemIndex(
    args.orderItemIndex,
  );
  const comment = args.comment.trim();

  if (
    args.evaluation !== "good" &&
    args.evaluation !== "disappointed"
  ) {
    throw new Error("evaluation is invalid");
  }

  if (!comment) {
    throw new Error("comment is empty");
  }

  return requestJson<AvatarReviewItem>(
    "/mall/me/avatar-reviews",
    {
      method: "POST",
      auth: "required",
      json: {
        orderId,
        orderItemIndex,
        evaluation: args.evaluation,
        comment,
      },
      unwrapData: true,
      messages: {
        requestErrorMessage:
          "createAvatarReview failed",
        nonJsonErrorMessage:
          "createAvatarReview failed: response is not json",
        invalidJsonErrorMessage:
          "createAvatarReview failed: invalid json",
      },
    },
  );
}