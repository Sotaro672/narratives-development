// frontend/console/shell/src/auth/domain/member.ts

/**
 * Backend /members/{id} 応答などに対応する Member DTO
 * - useAuth.ts の MemberDTO 相当
 */
export interface MemberDTO {
  /** メンバー ID（基本的には Firebase UID と同一） */
  id: string;

  /** 名（firstName） */
  firstName?: string | null;

  /** 姓（lastName） */
  lastName?: string | null;

  /** 名（かな） */
  firstNameKana?: string | null;

  /** 姓（かな） */
  lastNameKana?: string | null;

  /** メールアドレス */
  email?: string | null;

  /** 所属企業 ID */
  companyId: string;

  /**
   * 姓名を「姓 名」の形で結合した文字列
   * - firstName / lastName が両方空なら null
   */
  fullName?: string | null;
}
