// frontend/amol/src/features/resale/constants/resaleStatusOptions.ts

import {
  RESALE_STATUS_OPTIONS,
  type ResaleEditableStatus,
} from "../../shared/types/resale";

export {
  DEFAULT_RESALE_EDITABLE_STATUS,
  RESALE_STATUS_OPTIONS,
  type ResaleStatusOption,
} from "../../shared/types/resale";

/**
 * 値が利用者によって変更可能な再販ステータスか判定する。
 *
 * soldは売却処理によって設定されるため、
 * 編集可能なステータスには含めない。
 */
export function isResaleEditableStatus(value: unknown): value is ResaleEditableStatus {
  return typeof value === "string" && RESALE_STATUS_OPTIONS.some((option) => option.value === value);
}

/**
 * 編集可能な再販ステータスの表示名を返す。
 */
export function getResaleEditableStatusLabel(value: ResaleEditableStatus): string {
  return RESALE_STATUS_OPTIONS.find((option) => option.value === value)!.label;
}