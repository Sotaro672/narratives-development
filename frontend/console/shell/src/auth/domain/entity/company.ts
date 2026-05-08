// frontend/console/shell/src/auth/domain/company.ts

/**
 * Backend /companies/{id} 応答などに対応する Company DTO
 * - useAuth.ts の CompanyDTO 相当
 */
export interface CompanyDTO {
  /** 企業 ID */
  id?: string;

  /** 企業名 */
  name?: string;
}
