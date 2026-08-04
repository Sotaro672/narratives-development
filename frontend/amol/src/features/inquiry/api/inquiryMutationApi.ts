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
  InquiryReply,
  ReplyInquiryRequest,
} from "../../shared/types/inquiryTypes";

export async function createInquiry(
  payload: CreateInquiryRequest,
): Promise<Inquiry | null> {
  const json =
    await fetchInquiryWithAuth<
      ApiDataResponse<Inquiry>
    >(
      INQUIRY_BASE_PATH,
      {
        method: "POST",
        body: JSON.stringify(
          payload,
        ),
      },
    );

  return json.data ?? null;
}

export async function markInquiryAsRead(
  inquiryId: string,
): Promise<Inquiry | null> {
  const path =
    `${buildInquiryPath(
      inquiryId,
    )}/mark-as-read`;

  const json =
    await fetchInquiryWithAuth<
      ApiDataResponse<Inquiry>
    >(path, {
      method: "POST",
    });

  return json.data ?? null;
}

export async function replyInquiry(
  inquiryId: string,
  payload: ReplyInquiryRequest,
): Promise<InquiryReply | null> {
  const path =
    `${buildInquiryPath(
      inquiryId,
    )}/reply`;

  const json =
    await fetchInquiryWithAuth<
      ApiDataResponse<InquiryReply>
    >(path, {
      method: "POST",
      body: JSON.stringify(
        payload,
      ),
    });

  return json.data ?? null;
}

export async function closeInquiry(
  inquiryId: string,
): Promise<Inquiry | null> {
  const path =
    `${buildInquiryPath(
      inquiryId,
    )}/close`;

  const json =
    await fetchInquiryWithAuth<
      ApiDataResponse<Inquiry>
    >(path, {
      method: "POST",
    });

  return json.data ?? null;
}