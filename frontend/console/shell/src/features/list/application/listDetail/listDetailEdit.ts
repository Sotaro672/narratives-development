// frontend/console/shell/src/features/list/application/listDetail/listDetailEdit.ts

import type { ListStatus } from "../../../../shared/types/list";

export function normalizeListStatusForEdit(
  status: unknown,
): ListStatus {
  return status === "listing"
    ? "listing"
    : "suspended";
}