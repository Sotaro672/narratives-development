// frontend/amol/src/features/inquiry/api/inquiryListApi.ts

import type {
  ApiQueryParams,
} from "../../../lib/http";

import {
  fetchInquiryWithAuth,
  INQUIRY_BASE_PATH,
  type ApiItemsResponse,
  type ApiUnreadCountResponse,
} from "./inquiryHttpClient";

import type {
  GetUnreadInquiryCountParams,
  Inquiry,
  ListMeInquiriesParams,
  ListMeInquiriesResult,
} from "../../shared/types/inquiryTypes";

function normalizeQueryValue(
  value:
    | string
    | number
    | null
    | undefined,
): string | number | undefined {
  if (
    value === null ||
    value === undefined
  ) {
    return undefined;
  }

  if (typeof value === "number") {
    return Number.isFinite(value)
      ? value
      : undefined;
  }

  const normalized =
    value.trim();

  return normalized ||
    undefined;
}

function buildInquiryFilterQuery(
  params: GetUnreadInquiryCountParams,
): ApiQueryParams {
  return {
    productId:
      normalizeQueryValue(
        params.productId,
      ),

    status:
      normalizeQueryValue(
        params.status,
      ),

    inquiryType:
      normalizeQueryValue(
        params.inquiryType,
      ),

    searchQuery:
      normalizeQueryValue(
        params.searchQuery,
      ),
  };
}

function buildInquiryListQuery(
  params: ListMeInquiriesParams,
): ApiQueryParams {
  return {
    page:
      normalizeQueryValue(
        params.page,
      ),

    perPage:
      normalizeQueryValue(
        params.perPage,
      ),

    ...buildInquiryFilterQuery(
      params,
    ),
  };
}

export async function listMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  const json =
    await fetchInquiryWithAuth<
      ApiItemsResponse<Inquiry>
    >(
      INQUIRY_BASE_PATH,
      {
        method: "GET",
        signal: params.signal,
        query:
          buildInquiryListQuery(
            params,
          ),
      },
    );

  return {
    items: Array.isArray(
      json.items,
    )
      ? json.items
      : [],

    page: json.page,
    perPage: json.perPage,
  };
}

export async function getUnreadInquiryCount(
  params: GetUnreadInquiryCountParams = {},
): Promise<number> {
  const json =
    await fetchInquiryWithAuth<
      ApiUnreadCountResponse
    >(
      `${INQUIRY_BASE_PATH}/unread-count`,
      {
        method: "GET",
        query:
          buildInquiryFilterQuery(
            params,
          ),
      },
    );

  const unreadCount =
    Number(
      json.count ??
        json.unreadCount ??
        0,
    );

  if (
    !Number.isFinite(
      unreadCount,
    )
  ) {
    return 0;
  }

  return Math.max(
    0,
    Math.floor(
      unreadCount,
    ),
  );
}