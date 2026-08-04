// frontend/amol/src/features/inquiry/api/inquiryListApi.ts

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

function appendOptionalQuery(
  query: URLSearchParams,
  key: string,
  value:
    | string
    | number
    | null
    | undefined,
): void {
  if (
    value === null ||
    value === undefined
  ) {
    return;
  }

  const normalized =
    String(value).trim();

  if (!normalized) {
    return;
  }

  query.set(
    key,
    normalized,
  );
}

function appendInquiryFilters(
  query: URLSearchParams,
  params: GetUnreadInquiryCountParams,
): void {
  appendOptionalQuery(
    query,
    "productId",
    params.productId,
  );

  appendOptionalQuery(
    query,
    "status",
    params.status,
  );

  appendOptionalQuery(
    query,
    "inquiryType",
    params.inquiryType,
  );

  appendOptionalQuery(
    query,
    "searchQuery",
    params.searchQuery,
  );
}

export async function listMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  const query =
    new URLSearchParams();

  appendOptionalQuery(
    query,
    "page",
    params.page,
  );

  appendOptionalQuery(
    query,
    "perPage",
    params.perPage,
  );

  appendInquiryFilters(
    query,
    params,
  );

  const queryString =
    query.toString();

  const path = queryString
    ? `${INQUIRY_BASE_PATH}?${queryString}`
    : INQUIRY_BASE_PATH;

  const json =
    await fetchInquiryWithAuth<
      ApiItemsResponse<Inquiry>
    >(path, {
      method: "GET",
      signal: params.signal,
    });

  return {
    items: Array.isArray(
      json.items,
    )
      ? json.items
      : [],
    page: json.page,
    perPage: json.perPage,
    total: json.total,
    totalCount:
      json.totalCount,
  };
}

// ChatListPageなどから利用するための互換alias。
export async function fetchMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  return listMeInquiries(
    params,
  );
}

export async function getUnreadInquiryCount(
  params: GetUnreadInquiryCountParams = {},
): Promise<number> {
  const query =
    new URLSearchParams();

  appendInquiryFilters(
    query,
    params,
  );

  const queryString =
    query.toString();

  const path = queryString
    ? `${INQUIRY_BASE_PATH}/unread-count?${queryString}`
    : `${INQUIRY_BASE_PATH}/unread-count`;

  const json =
    await fetchInquiryWithAuth<
      ApiUnreadCountResponse
    >(path, {
      method: "GET",
    });

  const unreadCount = Number(
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