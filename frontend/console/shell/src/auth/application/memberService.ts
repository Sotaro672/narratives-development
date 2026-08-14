// frontend/console/shell/src/auth/application/memberService.ts
/// <reference types="vite/client" />

import type { MemberDTO } from "../../shared/types/member";
import {
  fetchCurrentMemberRaw,
  updateCurrentMemberProfileRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

export async function fetchCurrentMember(): Promise<MemberDTO | null> {
  const response = await fetchCurrentMemberRaw();
  return response as MemberDTO | null;
}

export type UpdateMemberProfileInput = {
  // PATCH /members/{docId} 用
  id: string;
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null;
};

type UpdateMemberProfilePayload = {
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null;
};

export async function updateCurrentMemberProfile(
  input: UpdateMemberProfileInput,
): Promise<MemberDTO | null> {
  const payload: UpdateMemberProfilePayload = {
    firstName: input.firstName,
    lastName: input.lastName,
    firstNameKana: input.firstNameKana,
    lastNameKana: input.lastNameKana,
  };

  if (input.email !== undefined) {
    payload.email = input.email;
  }

  const response = await updateCurrentMemberProfileRaw(input.id, payload);
  return response as MemberDTO | null;
}