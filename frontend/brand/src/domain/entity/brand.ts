// frontend/brand/src/domain/entity/brand.ts

/**
 * Brand
 * backend/internal/domain/brand/entity.go の Brand に対応するフロントエンド用ドメインモデル。
 *
 * - 日付は ISO8601 文字列として扱う
 * - フロント側では camelCase（websiteUrl / companyId など）を採用
 * - status は持たず isActive のみを利用
 */
export interface Brand {
  id: string;
  companyId: string;

  name: string;
  description: string;

  /** WebサイトURL。空文字 or undefined の場合は未設定扱い */
  websiteUrl?: string;

  /** 有効フラグ（status は持たない） */
  isActive: boolean;

  /** ブランド責任者 Member ID（任意） */
  managerId?: string | null;

  /** ブロックチェーン上のウォレットアドレス（必須） */
  walletAddress: string;

  /** 作成日時（ISO8601） */
  createdAt: string;
  /** 作成者（任意） */
  createdBy?: string | null;

  /** 更新日時（任意, ISO8601） */
  updatedAt?: string | null;
  /** 更新者（任意） */
  updatedBy?: string | null;

  /** 論理削除日時（任意, ISO8601） */
  deletedAt?: string | null;
  /** 論理削除者（任意） */
  deletedBy?: string | null;
}

/**
 * BrandPatch
 * 部分更新用パッチ（backend の BrandPatch に対応）
 * - undefined: 変更なし
 * - null: クリアを意味させたい場合に利用（実際の扱いはAPI側仕様に合わせる）
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
 * Brand ドメインの簡易バリデーション。
 * フロント側でフォーム送信前などに利用する想定（バックエンドの厳密検証とは独立）。
 */
export function validateBrand(b: Brand): string[] {
  const errors: string[] = [];

  if (!b.id?.trim()) errors.push("id is required");
  if (!b.companyId?.trim()) errors.push("companyId is required");
  if (!b.name?.trim()) errors.push("name is required");
  if (!b.description?.trim()) errors.push("description is required");
  if (!b.walletAddress?.trim()) errors.push("walletAddress is required");

  if (b.websiteUrl && b.websiteUrl.trim()) {
    try {
      const u = new URL(b.websiteUrl);
      if (!u.protocol || !u.host) {
        errors.push("websiteUrl must be a valid URL");
      }
    } catch {
      errors.push("websiteUrl must be a valid URL");
    }
  }

  return errors;
}

/**
 * Brand を「有効化」した新しいインスタンスを返すヘルパ。
 * （バックエンドの Activate/Deactivate と思想を揃えるクライアント側ユーティリティ）
 */
export function activateBrand(brand: Brand, updatedAt: string, updatedBy?: string | null): Brand {
  return {
    ...brand,
    isActive: true,
    updatedAt,
    updatedBy: updatedBy ?? brand.updatedBy ?? null,
  };
}

export function deactivateBrand(brand: Brand, updatedAt: string, updatedBy?: string | null): Brand {
  return {
    ...brand,
    isActive: false,
    updatedAt,
    updatedBy: updatedBy ?? brand.updatedBy ?? null,
  };
}
