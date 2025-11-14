// frontend/console/shell/src/auth/domain/authUser.ts
export interface AuthUser {
  uid: string;
  email: string | null;
  displayName: string | null;
  companyId: string | null;
  permissions: string[];
  assignedBrands: string[];
}
