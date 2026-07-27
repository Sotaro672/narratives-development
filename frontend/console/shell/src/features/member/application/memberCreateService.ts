// frontend/console/shell/src/features/member/application/memberCreateService.ts

import type {
  CreateMemberInput,
  Member,
  MemberStatus,
} from "../../../shared/types/member";

import { MemberRepositoryHTTP } from "../infrastructure/http/memberRepositoryHTTP";

const memberRepository = new MemberRepositoryHTTP();

// ─────────────────────────────────────────────
// Utility
// ─────────────────────────────────────────────

/**
 * カンマ区切りの文字列を文字列配列へ変換する。
 */
export const parseCommaSeparated = (
  value: string,
): string[] =>
  value
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item.length > 0);

// ─────────────────────────────────────────────
// Member作成
// ─────────────────────────────────────────────

export type CreateMemberParams = Omit<
  CreateMemberInput,
  "assignedBrands" | "status"
> & {
  /**
   * 割り当てるBrand ID。
   */
  assignedBrandIds: string[];

  /**
   * 未指定の場合はactiveを使用する。
   */
  status?: MemberStatus;
};

/**
 * Memberを作成する。
 *
 * Backend:
 * POST /members
 *
 * id・uid・companyId・作成日時などはBackend側で決定する。
 */
export async function createMember(
  params: CreateMemberParams,
): Promise<Member> {
  const firstName = params.firstName.trim();
  const lastName = params.lastName.trim();
  const firstNameKana =
    params.firstNameKana.trim();
  const lastNameKana =
    params.lastNameKana.trim();
  const email = params.email.trim();

  const permissions = params.permissions
    .map((permission) => permission.trim())
    .filter((permission) => permission.length > 0);

  const assignedBrands =
    params.assignedBrandIds
      .map((brandId) => brandId.trim())
      .filter((brandId) => brandId.length > 0);

  const status = params.status ?? "active";

  /**
   * MemberRepositoryHTTP.create()は書き込み可能な項目だけを
   * POST bodyへ変換する。
   *
   * Backendで生成される項目には初期値を設定し、
   * APIレスポンスとして返されたMemberを最終結果とする。
   */
  const member: Member = {
    id: "",
    uid: "",

    firstName,
    lastName,
    firstNameKana,
    lastNameKana,

    email,

    permissions,

    assignedBrands:
      assignedBrands.length > 0
        ? assignedBrands
        : null,

    companyId: "",
    status,

    createdAt: "",
    updatedAt: null,
    updatedBy: null,

    displayName: [
      lastName,
      firstName,
    ]
      .filter((value) => value.length > 0)
      .join(" "),
  };

  return memberRepository.create(member);
}