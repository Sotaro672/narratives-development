// frontend/console/shell/src/shared/types/member.ts

/**
 * Memberの正規型。
 *
 * GET /members
 * GET /members/me
 * GET /members/{uid}
 * POST /members
 * PATCH /members/{id}
 *
 * のレスポンスで共通利用する。
 */
export type Member = {
  /** FirestoreのMemberドキュメントID。 */
  id: string;

  /**
   * Firebase AuthenticationのUID。
   * 招待前のMemberでは空文字になる場合がある。
   */
  uid: string;

  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email: string;

  /** Permission名の配列。 */
  permissions: string[];

  /**
   * 割り当てられているBrand IDの配列。
   * nullは割り当てなしを表す。
   */
  assignedBrands: string[] | null;

  /** 所属会社ID。 */
  companyId: string;

  /** Backendから返されるMemberの状態。 */
  status: string;

  /** BackendからISO 8601形式の文字列として返される。 */
  createdAt: string;

  /** 未更新または値が存在しない場合はnull。 */
  updatedAt: string | null;

  /** 更新者ID。値が存在しない場合はnull。 */
  updatedBy: string | null;

  /** Backendで姓・名から生成される表示名。 */
  displayName: string;
};

/**
 * Member一覧取得時の検索条件。
 *
 * GET /members
 *
 * companyIdはFrontendから指定せず、
 * Backend側で認証中MemberのcompanyIdにスコープする。
 */
export type MemberFilter = {
  /** 名前・フリガナ・メールなどの検索文字列。 */
  searchQuery?: string;

  /** Firebase Authentication UID。 */
  uid?: string;

  /** 割り当てられているBrand ID。 */
  brandIds?: string[];

  /** Memberの状態。 */
  status?: string;
};

/**
 * Member作成時にFrontendから送信する項目。
 *
 * id、uid、companyId、日時、displayNameは
 * Backend側で決定されるため含めない。
 */
export type CreateMemberInput = {
  firstName: string;
  lastName: string;
  firstNameKana: string;
  lastNameKana: string;
  email: string;
  permissions: string[];
  assignedBrands: string[];
  status: string;
};

/**
 * Member更新時にFrontendから送信できる項目。
 *
 * undefinedは変更しないことを表す。
 * assignedBrandsの空配列は全解除を表す。
 */
export type MemberPatch = {
  firstName?: string;
  lastName?: string;
  firstNameKana?: string;
  lastNameKana?: string;
  email?: string;
  permissions?: string[];
  assignedBrands?: string[];
  status?: string;
};