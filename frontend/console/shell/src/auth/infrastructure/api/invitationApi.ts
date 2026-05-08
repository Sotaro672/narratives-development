// frontend/console/shell/src/auth/infrastructure/api/invitationApi.ts

// HTTP 実装は repository 層へ委譲
import type {
  InvitationInfo,
  ValidateResponse,
  CompanyResponse,
  BrandResponse,
  CompleteInvitationBackendPayload,
} from "../repository/invitationRepositoryHTTP";

import {
  fetchInvitationInfo as fetchInvitationInfoRepo,
  fetchCompanyNameById as fetchCompanyNameByIdRepo,
  fetchBrandNameById as fetchBrandNameByIdRepo,
  fetchBrandNamesByIds as fetchBrandNamesByIdsRepo,
  validateInvitation as validateInvitationRepo,
  completeInvitationOnBackend as completeInvitationOnBackendRepo,
} from "../repository/invitationRepositoryHTTP";

// ------------------------------
// 型の re-export
// ------------------------------
export type {
  InvitationInfo,
  ValidateResponse,
  CompanyResponse,
  BrandResponse,
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

export function fetchCompanyNameById(companyId: string): Promise<string> {
  return fetchCompanyNameByIdRepo(companyId);
}

export function fetchBrandNameById(brandId: string): Promise<string> {
  return fetchBrandNameByIdRepo(brandId);
}

export function fetchBrandNamesByIds(
  assignedBrandIds: string[],
): Promise<string[]> {
  return fetchBrandNamesByIdsRepo(assignedBrandIds);
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
