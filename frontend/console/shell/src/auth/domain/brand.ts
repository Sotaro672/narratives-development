// frontend/console/shell/src/auth/domain/brand.ts

/**
 * ブランド ID 型
 * - AuthUser.assignedBrands などで利用
 */
export type BrandId = string;

/**
 * 認証コンテキストや簡易表示で利用するブランド情報
 * （完全なブランドエンティティではなくサマリ用 DTO を想定）
 */
export interface BrandDTO {
  /** ブランド ID */
  id: BrandId;

  /** ブランド名 */
  name: string;

  /** 所属企業 ID（必要に応じて使用） */
  companyId?: string;

  /** 有効フラグなど、必要に応じて拡張 */
  isActive?: boolean;
}
