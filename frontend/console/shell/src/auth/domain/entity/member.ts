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
  companyId: string;
  fullName?: string | null;
}