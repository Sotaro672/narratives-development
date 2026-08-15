// frontend/console/shell/src/features/member/application/memberCreateService.ts

import type {
  CreateMemberInput,
  Member,
} from "../../../shared/types/member";

import { MemberRepositoryHTTP } from "../infrastructure/memberRepositoryHTTP";

const memberRepository = new MemberRepositoryHTTP();

// ─────────────────────────────────────────────
// Utility
// ─────────────────────────────────────────────

/**
 * カンマ区切りの文字列を文字列配列へ変換する。
 */
export const parseCommaSeparated = (value: string): string[] =>
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
  /** 割り当てるBrand ID。 */
  assignedBrandIds: string[];

  /** 未指定の場合はactiveを使用する。 */
  status?: string;
};

/**
 * Memberを作成する。
 *
 * Backend:
 * POST /members
 *
 * id・uid・companyId・作成日時・displayNameなどはBackend側で決定する。
 */
export async function createMember(
  params: CreateMemberParams,
): Promise<Member> {
  const input: CreateMemberInput = {
    firstName: params.firstName.trim(),
    lastName: params.lastName.trim(),
    firstNameKana: params.firstNameKana.trim(),
    lastNameKana: params.lastNameKana.trim(),
    email: params.email.trim(),
    permissions: params.permissions
      .map((permission) => permission.trim())
      .filter((permission) => permission.length > 0),
    assignedBrands: params.assignedBrandIds
      .map((brandId) => brandId.trim())
      .filter((brandId) => brandId.length > 0),
    status: params.status ?? "active",
  };

  return memberRepository.create(input);
}