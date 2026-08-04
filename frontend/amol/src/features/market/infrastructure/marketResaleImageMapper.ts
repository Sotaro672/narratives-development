// frontend/amol/src/features/market/infrastructure/marketResaleImageMapper.ts

import {
  isRecord,
} from "../../../components/utils/typeGuards";

import type {
  MarketResaleConditionImage,
  MarketResaleConditionImagesResponse,
} from "../types/marketResaleImage";

function normalizeString(
  value: unknown,
): string {
  if (typeof value !== "string") {
    return "";
  }

  return value.trim();
}

function isMarketResaleConditionImage(
  value: unknown,
): value is MarketResaleConditionImage {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    return false;
  }

  return (
    normalizeString(value.id) !== "" &&
    normalizeString(value.url) !== ""
  );
}

export function normalizeMarketResaleConditionImagesResponse(
  response: MarketResaleConditionImagesResponse,
): MarketResaleConditionImage[] {
  if (Array.isArray(response)) {
    return response.filter(
      isMarketResaleConditionImage,
    );
  }

  if (Array.isArray(response.data)) {
    return response.data.filter(
      isMarketResaleConditionImage,
    );
  }

  if (Array.isArray(response.items)) {
    return response.items.filter(
      isMarketResaleConditionImage,
    );
  }

  return [];
}