// frontend/console/shell/src/auth/domain/member.ts

export interface MemberDTO {
  /** Firestore members の docId */
  id: string;

  /** Firebase Auth UID */
  uid?: string | null;

  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;

  email?: string | null;

  /** 所属 companyId */
  companyId: string;

  /**
   * backend response の表示名。
   *
   * 正:
   * GET /members/{uid} response の displayName
   *
   * 例:
   * displayName: "あ い"
   */
  displayName?: string | null;
}