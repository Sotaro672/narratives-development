// frontend/amol/src/features/resale/constants/resaleStatusOptions.ts

import type {
  ResaleEditableStatus,
} from "../../shared/types/resale";

export type ResaleStatusOption = {
  value: ResaleEditableStatus;
  label: string;
};

export const RESALE_STATUS_OPTIONS:
  ResaleStatusOption[] = [
    {
      value: "listing",
      label: "出品中",
    },
    {
      value: "suspended",
      label: "公開停止",
    },
  ];

export const DEFAULT_RESALE_EDITABLE_STATUS:
  ResaleEditableStatus =
    "listing";

/**
 * 値が利用者によって変更可能な
 * 再販ステータスか判定する。
 *
 * soldは売却処理によって設定されるため、
 * 編集可能なステータスには含めない。
 */
export function isResaleEditableStatus(
  value: unknown,
): value is ResaleEditableStatus {
  return (
    typeof value === "string" &&
    RESALE_STATUS_OPTIONS.some(
      (option) =>
        option.value === value,
    )
  );
}

/**
 * 値を編集可能な再販ステータスへ正規化する。
 *
 * soldを含む編集対象外の値や不正な値の場合は、
 * デフォルトの出品中を返す。
 */
export function normalizeResaleEditableStatus(
  value: unknown,
): ResaleEditableStatus {
  return isResaleEditableStatus(value)
    ? value
    : DEFAULT_RESALE_EDITABLE_STATUS;
}

/**
 * 編集可能な再販ステータスの表示名を返す。
 */
export function getResaleEditableStatusLabel(
  value: ResaleEditableStatus,
): string {
  return (
    RESALE_STATUS_OPTIONS.find(
      (option) =>
        option.value === value,
    )?.label ??
    value
  );
}