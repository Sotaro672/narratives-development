// frontend/shell/src/shared/types/brand.ts

/**
 * Brand
 * backend/internal/domain/brand/entity.go に対応する共通型。
 *
 * - 日付は ISO8601 文字列
 * - status は持たず isActive のみを使用
 * - managerId, websiteUrl など一部は任意
 */
export interface Brand {
  id: string;
  companyId: string;

  name: string;
  description: string;

  /** 公式サイトURL。空文字 or undefined は未設定扱い */
  websiteUrl?: string;

  /** 有効フラグ（status 代替） */
  isActive: boolean;

  /** ブランド責任者 Member ID（任意） */
  managerId?: string | null;

  /** ブロックチェーン上のウォレットアドレス（必須） */
  walletAddress: string;

  /** 作成情報 */
  createdAt: string;
  createdBy?: string | null;

  /** 更新情報 */
  updatedAt?: string | null;
  updatedBy?: string | null;

  /** 論理削除情報 */
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * BrandPatch
 * 部分更新用（backend の BrandPatch に対応）
 * - undefined: 変更なし
 * - null: クリア（API仕様による）
 */
export interface BrandPatch {
  companyId?: string | null;
  name?: string | null;
  description?: string | null;
  websiteUrl?: string | null;
  isActive?: boolean | null;
  managerId?: string | null;
  walletAddress?: string | null;

  createdBy?: string | null;
  updatedAt?: string | null;
  updatedBy?: string | null;
  deletedAt?: string | null;
  deletedBy?: string | null;
}

/**
 * 共通ユーティリティ
 */

/** URL形式かどうか簡易チェック */
export function isValidUrl(value?: string): boolean {
  if (!value || !value.trim()) return true;
  try {
    const u = new URL(value);
    return Boolean(u.protocol && u.host);
  } catch {
    return false;
  }
}

/** フロント側簡易バリデーション */
export function validateBrand(b: Brand): string[] {
  const errors: string[] = [];

  if (!b.name?.trim()) errors.push("ブランド名は必須です");
  if (!b.description?.trim()) errors.push("ブランド説明は必須です");
  if (!b.walletAddress?.trim()) errors.push("ウォレットアドレスは必須です");

  if (b.websiteUrl && !isValidUrl(b.websiteUrl)) {
    errors.push("有効なURLを入力してください");
  }

  return errors;
}
