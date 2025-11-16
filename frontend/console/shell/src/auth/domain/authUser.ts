// frontend/console/shell/src/auth/domain/authUser.ts

export interface AuthUser {
  uid: string;

  /** Firebase Auth の email */
  email: string | null;

  /** 表示名（フルネームなど） */
  displayName: string | null;

  /** 所属企業 ID（useAuth.ts と同じく string 前提） */
  companyId: string;

  /** 権限一覧 */
  permissions: string[];

  /** 割り当てブランド */
  assignedBrands: string[];

  /** ▼ 追加：メンバー（currentMember）由来の氏名情報 */
  firstName?: string | null;
  lastName?: string | null;
  firstNameKana?: string | null;
  lastNameKana?: string | null;
}
