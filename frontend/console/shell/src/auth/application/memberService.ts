/// <reference types="vite/client" />

import type { MemberDTO } from "../domain/entity/member";
import {
  fetchCurrentMemberRaw,
  updateCurrentMemberProfileRaw,
} from "../infrastructure/repository/authRepositoryHTTP";

// -------------------------------
// 共通: 生 JSON → MemberDTO 変換
// -------------------------------
function mapRawToMemberDTO(
  raw: any,
  fallbackEmail?: string | null,
): MemberDTO {
  const firstName =
    raw?.firstName && String(raw.firstName).trim() !== ""
      ? String(raw.firstName).trim()
      : null;

  const lastName =
    raw?.lastName && String(raw.lastName).trim() !== ""
      ? String(raw.lastName).trim()
      : null;

  const firstNameKana =
    raw?.firstNameKana && String(raw.firstNameKana).trim() !== ""
      ? String(raw.firstNameKana).trim()
      : null;

  const lastNameKana =
    raw?.lastNameKana && String(raw.lastNameKana).trim() !== ""
      ? String(raw.lastNameKana).trim()
      : null;

  const displayNameFromResponse =
    raw?.displayName && String(raw.displayName).trim() !== ""
      ? String(raw.displayName).trim()
      : null;

  const displayNameFromNameParts =
    `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    // backend response の id は Firestore members の docId
    id: String(raw?.id ?? "").trim(),

    // Firebase Auth UID は backend response の uid を正とする
    uid: String(raw?.uid ?? "").trim(),

    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email: raw?.email ?? fallbackEmail ?? null,
    companyId: raw?.companyId ?? "",

    // backend response の displayName を正とする
    displayName: displayNameFromResponse ?? displayNameFromNameParts,
  };
}

// -------------------------------
// 現在メンバー取得
// -------------------------------
export async function fetchCurrentMember(): Promise<MemberDTO | null> {
  const raw = await fetchCurrentMemberRaw();
  if (!raw) return null;

  return mapRawToMemberDTO(raw);
}

// -------------------------------
// プロファイル更新
// -------------------------------
export type UpdateMemberProfileInput = {
  // PATCH /members/{docId} 用
  id: string;
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null;
};

export async function updateCurrentMemberProfile(
  input: UpdateMemberProfileInput,
): Promise<MemberDTO | null> {
  const payload: any = {
    firstName: input.firstName,
    lastName: input.lastName,
    firstNameKana: input.firstNameKana,
    lastNameKana: input.lastNameKana,
  };

  if (input.email !== undefined) {
    payload.email = input.email;
  }

  const raw = await updateCurrentMemberProfileRaw(input.id, payload);
  if (!raw) return null;

  return mapRawToMemberDTO(raw, input.email ?? null);
}