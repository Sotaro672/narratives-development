// frontend/company/src/domain/entity/company.ts

/**
 * Company
 * backend/internal/domain/company/entity.go の Company に対応。
 *
 * - id       : 企業ID
 * - name     : 企業名
 * - admin    : root 権限を持つ memberId
 * - isActive : 有効フラグ
 *
 * 日付はフロントでは ISO8601 文字列 (例: "2025-01-10T00:00:00Z") として扱う。
 * deletedAt / deletedBy は省略または null 許容（未削除時）。
 */
export interface Company {
  id: string;
  name: string;
  admin: string; // root権限を持ったmemberId
  isActive: boolean;

  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * Company の簡易バリデーション
 * backend/internal/domain/company/entity.go の validate() / validateUpdateOnly()
 * と整合する範囲でフロント側チェックを行う。
 *
 * 厳密な検証（特に updatedAt / deletedAt の時系列整合など）は backend に委譲する。
 */
export function validateCompany(c: Company): boolean {
  // id, name, admin
  if (!c.id || !c.id.trim()) return false;
  if (!c.name || !c.name.trim()) return false;
  if (!c.admin || !c.admin.trim()) return false;

  // createdBy / updatedBy
  if (!c.createdBy || !c.createdBy.trim()) return false;
  if (!c.updatedBy || !c.updatedBy.trim()) return false;

  // createdAt / updatedAt (フォーマット検証)
  if (!c.createdAt || Number.isNaN(Date.parse(c.createdAt))) return false;
  if (!c.updatedAt || Number.isNaN(Date.parse(c.updatedAt))) return false;

  const createdAt = new Date(c.createdAt).getTime();
  const updatedAt = new Date(c.updatedAt).getTime();
  if (updatedAt < createdAt) return false;

  // deletedAt がある場合は形式のみ確認（順序は backend に委譲）
  if (
    c.deletedAt != null &&
    c.deletedAt !== "" &&
    Number.isNaN(Date.parse(c.deletedAt))
  ) {
    return false;
  }

  // deletedBy は deletedAt がある場合のみ有効であるべきだが、
  // 厳密な整合性チェックは backend 側ルールに委譲。
  if (c.deletedBy != null && c.deletedBy !== "" && !c.deletedBy.trim()) {
    return false;
  }

  return true;
}

/**
 * GraphQL / フォーム入力用 DTO
 * - 新規作成・更新時に利用する軽量型
 * - createdAt/updatedAt は通常 backend 側で付与する想定
 */
export interface CompanyInput {
  id?: string;
  name: string;
  admin: string;
  isActive?: boolean;
}
