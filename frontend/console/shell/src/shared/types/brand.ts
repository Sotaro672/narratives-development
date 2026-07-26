// frontend/console/shell/src/shared/types/brand.ts

/**
 * Brand
 * backend/internal/adapters/in/http/console/handler/brand_handler.go の
 * brandDTO に対応する共通型。
 *
 * Brand 型はこのファイルのみを正規定義とし、
 * features/brand や auth 配下では再定義しない。
 *
 * - 日付は ISO 8601 文字列として扱う
 * - status は持たず isActive を使用する
 * - memberName は一覧・詳細表示用の派生値
 */
export interface Brand {
  id: string;
  companyId: string;

  name: string;
  description: string;

  /** 公式WebサイトURL。未設定の場合は空文字または undefined */
  websiteUrl?: string;

  /** ブランドアイコン画像のURL。未設定の場合は空文字または undefined */
  brandIcon?: string;

  /** ブランド背景画像のURL。未設定の場合は空文字または undefined */
  brandBackgroundImage?: string;

  /** ブランドの有効状態 */
  isActive: boolean;

  /** ブランド責任者のMember ID */
  managerId?: string | null;

  /** ブランド責任者の表示名。APIレスポンス表示用 */
  memberName?: string | null;

  /** ブロックチェーン上のウォレットアドレス */
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
 * PATCH /brands/{id} のリクエストボディに対応する部分更新型。
 *
 * - undefined: リクエストへ含めず、現在値を変更しない
 * - null: API側で許可されている項目のみクリア値として扱う
 * - memberName は表示用の派生値なので更新対象に含めない
 */
export interface BrandPatch {
  name?: string | null;
  description?: string | null;
  websiteUrl?: string | null;
  brandIcon?: string | null;
  brandBackgroundImage?: string | null;
  isActive?: boolean | null;
  managerId?: string | null;
}

/** URL形式かどうかを簡易確認する。空値は未設定として許可する。 */
export function isValidUrl(value?: string | null): boolean {
  if (value === undefined || value === null || value === "") {
    return true;
  }

  try {
    const url = new URL(value);
    return Boolean(url.protocol && url.host);
  } catch {
    return false;
  }
}

/**
 * Brandのフロントエンド側簡易バリデーション。
 * trimによる補正は行わず、入力値をそのまま判定する。
 */
export function validateBrand(brand: Brand): string[] {
  const errors: string[] = [];

  if (!brand.id) {
    errors.push("id is required");
  }

  if (!brand.companyId) {
    errors.push("companyId is required");
  }

  if (!brand.name) {
    errors.push("name is required");
  }

  if (!brand.description) {
    errors.push("description is required");
  }

  if (!brand.walletAddress) {
    errors.push("walletAddress is required");
  }

  if (!isValidUrl(brand.websiteUrl)) {
    errors.push("websiteUrl must be a valid URL");
  }

  return errors;
}

/** Brandを有効化した新しいオブジェクトを返す。 */
export function activateBrand(
  brand: Brand,
  updatedAt: string,
  updatedBy?: string | null,
): Brand {
  return {
    ...brand,
    isActive: true,
    updatedAt,
    updatedBy: updatedBy ?? brand.updatedBy ?? null,
  };
}

/** Brandを無効化した新しいオブジェクトを返す。 */
export function deactivateBrand(
  brand: Brand,
  updatedAt: string,
  updatedBy?: string | null,
): Brand {
  return {
    ...brand,
    isActive: false,
    updatedAt,
    updatedBy: updatedBy ?? brand.updatedBy ?? null,
  };
}