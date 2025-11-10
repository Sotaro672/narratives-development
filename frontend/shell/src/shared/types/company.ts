// frontend/shell/src/shared/types/company.ts

/**
 * Company
 * backend/internal/domain/company/entity.go に対応。
 *
 * - 各フィールドの意味:
 *   - id: 企業ID（UUID想定）
 *   - name: 企業名
 *   - admin: root権限を持つメンバーのID
 *   - isActive: 現在アクティブかどうか
 *   - createdAt / updatedAt / deletedAt: ISO8601 UTC 文字列
 *   - createdBy / updatedBy / deletedBy: 操作したユーザーのID
 */
export interface Company {
  id: string;
  name: string;
  admin: string;
  isActive: boolean;
  createdAt: string;
  createdBy: string;
  updatedAt: string;
  updatedBy: string;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * CompanyInput
 * GraphQLやフォーム入力用のDTO。
 * - createdAt / updatedAt はサーバー側で設定されるため不要。
 */
export interface CompanyInput {
  name: string;
  admin: string;
  isActive?: boolean;
}

/**
 * Companyの妥当性チェック
 * backend/internal/domain/company/entity.go の validate() に準拠。
 */
export function validateCompany(c: Company): boolean {
  if (!c.id?.trim()) return false;
  if (!c.name?.trim()) return false;
  if (!c.admin?.trim()) return false;
  if (!c.createdBy?.trim()) return false;
  if (!c.updatedBy?.trim()) return false;

  if (!c.createdAt || Number.isNaN(Date.parse(c.createdAt))) return false;
  if (!c.updatedAt || Number.isNaN(Date.parse(c.updatedAt))) return false;

  const created = new Date(c.createdAt).getTime();
  const updated = new Date(c.updatedAt).getTime();
  if (updated < created) return false;

  if (c.deletedAt && Number.isNaN(Date.parse(c.deletedAt))) return false;
  if (c.deletedBy != null && c.deletedBy !== "" && !c.deletedBy.trim()) return false;

  return true;
}
