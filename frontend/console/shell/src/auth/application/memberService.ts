//frontend\console\shell\src\auth\application\memberService.ts
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
  fallbackUid: string,
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
    // backend response の id は Firestore members の docId
    id: String(raw?.id ?? "").trim(),

    // Firebase Auth UID
    // backend が uid を返していればそれを使い、無ければ fetchCurrentMember(uid) の uid で補完する
    uid: String(raw?.uid ?? fallbackUid ?? "").trim(),

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
  const firebaseUid = String(uid ?? "").trim();
  if (!firebaseUid) return null;

  const raw = await fetchCurrentMemberRaw(firebaseUid);
  if (!raw) return null;

  return mapRawToMemberDTO(raw, firebaseUid);
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

  return mapRawToMemberDTO(raw, String(raw?.uid ?? ""), input.email ?? null);
}