// frontend/console/shell/src/auth/domain/auth.ts

/**
 * 認証済みユーザー情報（Firebase Auth + Firestore/users/{uid} などの統合ビュー）
 */
export interface Auth {
  /** Firebase Auth の UID */
  uid: string;

  /** Firebase Auth の email */
  email: string | null;

  /** 表示名（フルネームなど） */
  displayName: string | null;

  /**
   * 所属企業 ID
   * - Firestore users/{uid}.companyId 由来
   * - まだ会社未作成の場合などは null / undefined の可能性あり
   */
  companyId?: string | null;

  /** 権限一覧（Permission.name の配列） */
  permissions: string[];

  /** 割り当てブランド ID の配列 */
  assignedBrands: string[];

  /** ▼ currentMember 由来の氏名情報（任意） */
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
}
