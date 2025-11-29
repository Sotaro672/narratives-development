//frontend\console\shell\src\auth\application\memberService.ts
/// <reference types="vite/client" />

// frontend/console/shell/src/auth/application/memberService.ts

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
  fallbackId: string,
  fallbackEmail?: string | null,
): MemberDTO {
  const firstName =
    raw?.firstName && String(raw.firstName).trim() !== ""
      ? String(raw.firstName)
      : null;
  const lastName =
    raw?.lastName && String(raw.lastName).trim() !== ""
      ? String(raw.lastName)
      : null;

  const firstNameKana =
    raw?.firstNameKana && String(raw.firstNameKana).trim() !== ""
      ? String(raw.firstNameKana)
      : null;

  const lastNameKana =
    raw?.lastNameKana && String(raw.lastNameKana).trim() !== ""
      ? String(raw.lastNameKana)
      : null;

  const full = `${lastName ?? ""} ${firstName ?? ""}`.trim() || null;

  return {
    id: raw?.id ?? fallbackId,
    firstName,
    lastName,
    firstNameKana,
    lastNameKana,
    email: raw?.email ?? fallbackEmail ?? null,
    companyId: raw?.companyId ?? "",
    fullName: full,
  };
}

// -------------------------------
// 現在メンバー取得
// -------------------------------

export async function fetchCurrentMember(uid: string): Promise<MemberDTO | null> {
  const raw = await fetchCurrentMemberRaw(uid);
  if (!raw) return null;

  return mapRawToMemberDTO(raw, uid);
}

// -------------------------------
// プロファイル更新
// -------------------------------

// ★ email を含められるようにする
export type UpdateMemberProfileInput = {
  id: string;
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email?: string | null; // ← 追加
};

export async function updateCurrentMemberProfile(
  input: UpdateMemberProfileInput,
): Promise<MemberDTO | null> {
  // PATCH の payload（HTTP レイヤは payload の中身を知らない）
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

  return mapRawToMemberDTO(raw, input.id, input.email ?? null);
}
