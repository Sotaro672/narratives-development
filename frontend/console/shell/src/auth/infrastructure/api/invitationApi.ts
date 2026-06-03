// frontend/console/shell/src/auth/infrastructure/api/invitationApi.ts

// HTTP 実装は repository 層へ委譲
import type {
  InvitationInfo,
  ValidateResponse,
  CompleteInvitationBackendPayload,
} from "../repository/invitationRepositoryHTTP";

import {
  fetchInvitationInfo as fetchInvitationInfoRepo,
  validateInvitation as validateInvitationRepo,
  completeInvitationOnBackend as completeInvitationOnBackendRepo,
} from "../repository/invitationRepositoryHTTP";

// ------------------------------
// 型の re-export
// ------------------------------
export type {
  InvitationInfo,
  ValidateResponse,
  CompleteInvitationBackendPayload,
};

// ------------------------------
// 関数のラッパー / re-export
// （既存コードからの import パスを変えずに使えるようにする）
// ------------------------------

export function fetchInvitationInfo(
  token: string,
): Promise<InvitationInfo> {
  return fetchInvitationInfoRepo(token);
}

export function validateInvitation(
  token: string,
): Promise<ValidateResponse> {
  return validateInvitationRepo(token);
}

export function completeInvitationOnBackend(
  payload: CompleteInvitationBackendPayload,
): Promise<void> {
  return completeInvitationOnBackendRepo(payload);
}