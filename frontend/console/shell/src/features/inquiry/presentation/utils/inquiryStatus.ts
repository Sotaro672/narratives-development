// frontend/console/shell/src/features/inquiry/presentation/utils/inquiryStatus.ts

import type {
  InquiryStatus,
} from "../../../../shared/types/inquiry";

export type InquiryStatusValue =
  | InquiryStatus
  | null
  | undefined;

export type InquiryStatusButtonVariant =
  | "danger"
  | "neutral";

function normalizeInquiryStatus(
  value: InquiryStatusValue,
): string {
  return String(value ?? "").trim();
}

/**
 * 問い合わせステータスの表示名を返す。
 *
 * 一覧画面では未読状態を優先して表示するため、
 * isReadを指定できるようにする。
 */
export function getInquiryStatusLabel(
  statusValue: InquiryStatusValue,
  isRead?: boolean | null,
): string {
  if (isRead === false) {
    return "未読";
  }

  const status =
    normalizeInquiryStatus(statusValue);

  switch (status) {
    case "open":
      return "未対応";

    case "in_progress":
      return "対応中";

    case "resolved":
      return "対応済み";

    case "closed":
      return "クローズ";

    default:
      return status || "-";
  }
}

/**
 * 問い合わせが対応済みか判定する。
 */
export function isResolvedStatus(
  statusValue: InquiryStatusValue,
): boolean {
  return (
    normalizeInquiryStatus(statusValue) ===
    "resolved"
  );
}

/**
 * 問い合わせがクローズ済みか判定する。
 */
export function isClosedStatus(
  statusValue: InquiryStatusValue,
): boolean {
  return (
    normalizeInquiryStatus(statusValue) ===
    "closed"
  );
}

/**
 * 問い合わせが未対応状態か判定する。
 */
export function isUnresolvedStatus(
  statusValue: InquiryStatusValue,
): boolean {
  return (
    normalizeInquiryStatus(statusValue) ===
    "open"
  );
}

/**
 * PageStyleのステータスボタン表示種別を返す。
 */
export function getInquiryStatusButtonVariant(
  statusValue: InquiryStatusValue,
): InquiryStatusButtonVariant {
  return isUnresolvedStatus(statusValue)
    ? "danger"
    : "neutral";
}