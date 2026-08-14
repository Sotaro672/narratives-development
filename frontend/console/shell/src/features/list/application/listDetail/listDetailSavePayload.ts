// frontend/console/shell/src/features/list/application/listDetail/listDetailSavePayload.ts

import type { ListStatus } from "../../../../shared/types/list";

export type ListDetailSavePayload = {
  title?: string;
  description?: string;
  status?: ListStatus;
  assigneeId?: string;
};