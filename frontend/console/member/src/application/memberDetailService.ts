// frontend/console/member/src/application/memberDetailService.ts

import type { Member } from "../domain/entity/member";
import { auth } from "../../../shell/src/auth/infrastructure/config/firebaseClient";
import { API_BASE } from "./memberListService";
import { fetchMemberByIdWithToken } from "../infrastructure/query/memberQuery";

/**
 * メンバー詳細取得
 * - /members/:id を叩いて Member を取得
 * - 姓名が空の場合も ID にフォールバックしない
 */
export async function fetchMemberDetail(memberId: string): Promise<Member | null> {
  const id = String(memberId ?? "").trim();
  if (!id) return null;

  const currentUser = auth.currentUser;
  if (!currentUser) {
    throw new Error("未認証のためメンバー情報を取得できません。");
  }

  const token = await currentUser.getIdToken();

  console.log("[memberDetailService.fetchMemberDetail] GET", `${API_BASE}/members/${id}`);

  const raw = await fetchMemberByIdWithToken(token, id);
  if (!raw) return null;

  const noFirst =
    raw.firstName === null ||
    raw.firstName === undefined ||
    raw.firstName === "";

  const noLast =
    raw.lastName === null ||
    raw.lastName === undefined ||
    raw.lastName === "";

  return {
    ...raw,
    id: raw.id ?? id,
    firstName: noFirst ? null : raw.firstName ?? null,
    lastName: noLast ? null : raw.lastName ?? null,
  } as Member;
}
