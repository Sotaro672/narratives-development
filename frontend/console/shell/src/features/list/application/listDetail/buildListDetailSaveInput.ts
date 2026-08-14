// frontend/console/shell/src/features/list/application/listDetail/buildListDetailSaveInput.ts

import type { ListStatus } from "../../../../shared/types/list";
import type { ListDetailSavePayload } from "./listDetailSavePayload";

export type BuildListDetailSaveInputArgs = {
  payload?: ListDetailSavePayload;
  draftListingTitle: string;
  draftDescription: string;
  draftStatus: ListStatus;
  draftAssigneeId: string;
  currentUserUid?: string | null;
};

export type BuiltListDetailSaveInput = {
  title: string;
  description: string;
  status: ListStatus;
  assigneeId?: string;
  updatedBy: string;
};

export function buildListDetailSaveInput(
  args: BuildListDetailSaveInputArgs,
): BuiltListDetailSaveInput {
return {
  title: args.payload?.title ?? args.draftListingTitle,
  description: args.payload?.description ?? args.draftDescription,
  status: args.payload?.status ?? args.draftStatus,
  assigneeId:
    (args.payload?.assigneeId ?? args.draftAssigneeId) || undefined,
  updatedBy: args.currentUserUid ?? "system",
};
}