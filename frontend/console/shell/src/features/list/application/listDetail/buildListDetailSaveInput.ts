// frontend/console/shell/src/features/list/application/listDetail/buildListDetailSaveInput.ts

import {
  isValidListStatus,
  type ListStatus,
} from "../../../../shared/types/list";

import type { ListDetailSavePayload } from "./listDetailSavePayload";

export type BuildListDetailSaveInputArgs = {
  payload?: ListDetailSavePayload;

  draftListingTitle: string;
  draftDescription: string;
  draftStatus: ListStatus;
  draftAssigneeId: string;

  currentAssigneeId?: string | null;
  currentUserUid?: unknown;
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
  const title =
    String(args.payload?.title ?? "").trim() ||
    String(args.payload?.listingTitle ?? "").trim() ||
    String(args.draftListingTitle ?? "").trim();

  const description =
    args.payload &&
    args.payload.description !== undefined
      ? String(args.payload.description ?? "")
      : String(args.draftDescription ?? "");

  const payloadStatus =
    String(args.payload?.status ?? "").trim();

  const status =
    isValidListStatus(payloadStatus)
      ? payloadStatus
      : args.draftStatus;

  const assigneeId =
    String(args.payload?.assigneeId ?? "").trim() ||
    String(args.draftAssigneeId ?? "").trim() ||
    String(args.currentAssigneeId ?? "").trim() ||
    undefined;

  const updatedBy =
    String(args.currentUserUid ?? "").trim() ||
    "system";

  return {
    title,
    description,
    status,
    assigneeId,
    updatedBy,
  };
}