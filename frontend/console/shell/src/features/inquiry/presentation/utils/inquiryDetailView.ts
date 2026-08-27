// frontend/console/shell/src/features/inquiry/presentation/utils/inquiryDetailView.ts

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import type {
  InquiryDetail,
  InquiryOrderItemSummary,
  InquiryOrderSummary,
} from "../../../../shared/types/inquiry";

export type InquiryReplyView =
  InquiryDetail["replies"][number];

export function textOrDash(
  value: string | null | undefined,
): string {
  const normalized = String(value ?? "").trim();

  return normalized || "-";
}

export function normalizeText(
  value: unknown,
): string {
  return String(value ?? "").trim();
}

export function uniqueTextValues(
  values: Array<string | null | undefined>,
): string[] {
  const seen = new Set<string>();
  const result: string[] = [];

  for (const value of values) {
    const normalized =
      normalizeText(value);

    if (
      !normalized ||
      normalized === "-"
    ) {
      continue;
    }

    if (seen.has(normalized)) {
      continue;
    }

    seen.add(normalized);
    result.push(normalized);
  }

  return result;
}

export function getOrderTransferredAtLabel(
  order: InquiryOrderSummary,
): string {
  if (order.items.length === 0) {
    return "-";
  }

  const transferredAtValues =
    uniqueTextValues(
      order.items.map(
        (
          item:
            InquiryOrderItemSummary,
        ) =>
          item.transferredAt ?? null,
      ),
    );

  if (
    transferredAtValues.length === 0
  ) {
    return "-";
  }

  return transferredAtValues
    .map((transferredAt) =>
      safeDateTimeLabelJa(
        transferredAt,
        "-",
      ),
    )
    .join(" / ");
}

export function getReplySenderLabel(
  reply: InquiryReplyView,
  memberId: string,
): string {
  switch (reply.senderType) {
    case "member":
      return reply.senderId === memberId
        ? "自分"
        : "担当者";

    case "system":
      return "AMOL";

    case "avatar":
      return "お客様";

    default:
      return "-";
  }
}

export function isReplyFromCurrentMember(
  reply: InquiryReplyView,
  memberId: string,
): boolean {
  return (
    reply.senderType === "member" &&
    reply.senderId === memberId
  );
}