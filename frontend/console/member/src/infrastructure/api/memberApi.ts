// frontend/member/src/infrastructure/api/memberApi.ts

export type CreateMemberRequest = {
  id?: string;
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string | null;
  permissions: string[];
  assignedBrands?: string[] | null;
};

export type MemberResponse = {
  /**
   * Firestore member document ID
   */
  id: string;

  /**
   * Firebase Auth UID
   * GET /members/{uid} にはこちらを渡す
   */
  uid?: string;

  firstName?: string;
  lastName?: string;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
  email?: string;
  permissions: string[];
  assignedBrands?: string[];
  companyId?: string;
  status?: string;
  displayName?: string;
  createdAt: string;
  updatedAt?: string | null;
};