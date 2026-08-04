// frontend/amol/src/features/market/infrastructure/marketReviewMapper.ts

import {
  isFiniteNumber,
  isRecord,
} from "../../../components/utils/typeGuards";

import type {
  ProductBlueprintReview,
  ProductBlueprintReviewPage,
} from "../../shared/types/review";

import type {
  MarketProductBlueprintReviewsResponse,
} from "../../shared/types/marketReview";

function normalizeString(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

function toFiniteNumber(
  value: unknown,
  fallback = 0,
): number {
  if (isFiniteNumber(value)) {
    return value;
  }

  if (
    typeof value === "string" &&
    value.trim() !== ""
  ) {
    const parsed = Number(value);

    return isFiniteNumber(parsed)
      ? parsed
      : fallback;
  }

  return fallback;
}

function toBoolean(
  value: unknown,
): boolean {
  return value === true;
}

function normalizeReview(
  value: unknown,
): ProductBlueprintReview | null {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return null;
  }

  const id =
    normalizeString(value.id);

  if (!id) {
    return null;
  }

  return {
    id,
    productBlueprintId:
      normalizeString(
        value.productBlueprintId,
      ),
    avatarId:
      normalizeString(
        value.avatarId,
      ),
    avatarName:
      normalizeString(
        value.avatarName,
      ),
    avatarIcon:
      normalizeString(
        value.avatarIcon,
      ),
    rating:
      toFiniteNumber(
        value.rating,
      ),
    title:
      normalizeString(
        value.title,
      ),
    body:
      normalizeString(
        value.body,
      ),
    helpfulVotes:
      toFiniteNumber(
        value.helpfulVotes,
      ),
    totalVotes:
      toFiniteNumber(
        value.totalVotes,
      ),
    reviewedAt:
      normalizeString(
        value.reviewedAt ||
          value.createdAt,
      ),
    status:
      normalizeString(
        value.status,
      ),
  };
}

export function normalizeMarketProductBlueprintReviewsResponse(
  response: MarketProductBlueprintReviewsResponse,
  fallbackPage: number,
  fallbackPerPage: number,
): ProductBlueprintReviewPage {
  const root: Record<string, unknown> =
    isRecord(response) &&
    !Array.isArray(response)
      ? response
      : {};

  const rawData =
    root["data"];

  const data:
    Record<string, unknown> | null =
      isRecord(rawData) &&
      !Array.isArray(rawData)
        ? rawData
        : null;

  const source:
    Record<string, unknown> =
      data ?? root;

  const rawItems: unknown[] =
    Array.isArray(
      source["items"],
    )
      ? source["items"]
      : Array.isArray(
            source["reviews"],
          )
        ? source["reviews"]
        : [];

  const items = rawItems
    .map(
      (item: unknown) =>
        normalizeReview(item),
    )
    .filter(
      (
        item:
          | ProductBlueprintReview
          | null,
      ): item is ProductBlueprintReview =>
        item !== null,
    );

  return {
    items,
    page:
      toFiniteNumber(
        source["page"],
        fallbackPage,
      ) ||
      fallbackPage,
    perPage:
      toFiniteNumber(
        source["perPage"],
        fallbackPerPage,
      ) ||
      fallbackPerPage,
    total:
      toFiniteNumber(
        source["total"] ??
          source["totalCount"],
        items.length,
      ),
    hasNext:
      toBoolean(
        source["hasNext"],
      ),
  };
}