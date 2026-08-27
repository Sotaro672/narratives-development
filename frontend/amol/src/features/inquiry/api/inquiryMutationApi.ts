// frontend/amol/src/features/inquiry/api/inquiryMutationApi.ts

import {
  buildInquiryPath,
  fetchInquiryWithAuth,
  INQUIRY_BASE_PATH,
  type ApiDataResponse,
} from "./inquiryHttpClient";

import type {
  CreateInquiryRequest,
  Inquiry,
  InquiryDetail,
  InquiryReply,
  ReplyInquiryRequest,
} from "../../shared/types/inquiryTypes";

export async function createInquiry(
  payload: CreateInquiryRequest,
): Promise<Inquiry> {
  const json = await fetchInquiryWithAuth<ApiDataResponse<Inquiry>>(
    INQUIRY_BASE_PATH,
    {
      method: "POST",
      json: payload,
    },
  );

  return json.data;
}

export async function markInquiryAsRead(
  inquiryId: string,
): Promise<InquiryDetail> {
  const path = `${buildInquiryPath(inquiryId)}/mark-as-read`;

  const json = await fetchInquiryWithAuth<ApiDataResponse<InquiryDetail>>(
    path,
    {
      method: "POST",
    },
  );

  return json.data;
}

export async function replyInquiry(
  inquiryId: string,
  payload: ReplyInquiryRequest,
): Promise<InquiryReply> {
  const path = `${buildInquiryPath(inquiryId)}/reply`;

  const json = await fetchInquiryWithAuth<ApiDataResponse<InquiryReply>>(
    path,
    {
      method: "POST",
      json: payload,
    },
  );

  return json.data;
}

export async function closeInquiry(
  inquiryId: string,
): Promise<InquiryDetail> {
  const path = `${buildInquiryPath(inquiryId)}/close`;

  const json = await fetchInquiryWithAuth<ApiDataResponse<InquiryDetail>>(
    path,
    {
      method: "POST",
    },
  );

  return json.data;
}