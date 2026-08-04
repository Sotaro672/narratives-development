// frontend/amol/src/features/inquiry/api/inquiryThreadApi.ts

import {
  buildInquiryPath,
  fetchInquiryWithAuth,
  type ApiDataResponse,
  type ApiItemsResponse,
} from "./inquiryHttpClient";

import type {
  Inquiry,
  InquiryReply,
  InquiryThread,
} from "../../shared/types/inquiryTypes";

export async function getInquiry(
  inquiryId: string,
): Promise<Inquiry | null> {
  const json =
    await fetchInquiryWithAuth<
      ApiDataResponse<Inquiry>
    >(
      buildInquiryPath(
        inquiryId,
      ),
      {
        method: "GET",
      },
    );

  return json.data ?? null;
}

export async function listInquiryReplies(
  inquiryId: string,
): Promise<InquiryReply[]> {
  const path =
    `${buildInquiryPath(
      inquiryId,
    )}/replies`;

  const json =
    await fetchInquiryWithAuth<
      ApiItemsResponse<InquiryReply>
    >(path, {
      method: "GET",
    });

  return Array.isArray(
    json.items,
  )
    ? json.items
    : [];
}

export async function getInquiryThread(
  inquiryId: string,
): Promise<InquiryThread> {
  const [
    inquiry,
    replies,
  ] = await Promise.all([
    getInquiry(
      inquiryId,
    ),
    listInquiryReplies(
      inquiryId,
    ),
  ]);

  return {
    inquiry,
    replies,
  };
}