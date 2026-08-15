// frontend/amol/src/features/inquiry/api/inquiryListApi.ts

import type { ApiQueryParams } from "../../../lib/http";

import {
  fetchInquiryWithAuth,
  INQUIRY_BASE_PATH,
  type ApiPagedItemsResponse,
  type ApiUnreadCountResponse,
} from "./inquiryHttpClient";

import type {
  GetUnreadInquiryCountParams,
  InquiryListItem,
  ListMeInquiriesParams,
  ListMeInquiriesResult,
} from "../../shared/types/inquiryTypes";

function buildInquiryFilterQuery(
  params: GetUnreadInquiryCountParams,
): ApiQueryParams {
  return {
    productId: params.productId,
    status: params.status,
    inquiryType: params.inquiryType,
    searchQuery: params.searchQuery,
  };
}

function buildInquiryListQuery(
  params: ListMeInquiriesParams,
): ApiQueryParams {
  return {
    page: params.page,
    perPage: params.perPage,
    ...buildInquiryFilterQuery(params),
  };
}

export async function listMeInquiries(
  params: ListMeInquiriesParams = {},
): Promise<ListMeInquiriesResult> {
  const json = await fetchInquiryWithAuth<
    ApiPagedItemsResponse<InquiryListItem>
  >(INQUIRY_BASE_PATH, {
    method: "GET",
    signal: params.signal,
    query: buildInquiryListQuery(params),
  });

  return {
    items: json.items,
    page: json.page,
    perPage: json.perPage,
  };
}

export async function getUnreadInquiryCount(
  params: GetUnreadInquiryCountParams = {},
): Promise<number> {
  const json = await fetchInquiryWithAuth<ApiUnreadCountResponse>(
    `${INQUIRY_BASE_PATH}/unread-count`,
    {
      method: "GET",
      query: buildInquiryFilterQuery(params),
    },
  );

  return json.unreadCount;
}