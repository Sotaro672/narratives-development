// frontend/console/shell/src/features/member/domain/repository/memberRepository.ts

import type {
  Member,
  MemberPatch,
} from "../../../../shared/types/member";
import type {
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../../../shared/types/common/common";

/**
 * MemberFilter
 * backend/internal/domain/member/repository_port.go の Filter に対応。
 *
 * - 日付は ISO 8601 文字列
 * - undefined は「条件指定なし」
 * - companyIdは指定しない
 * - Backend側でCurrentMemberのcompanyIdに必ずスコープされる
 */
export interface MemberFilter {
  /** 名前・フリガナ・メールなどの部分一致検索 */
  searchQuery?: string;

  /**
   * Firebase Authentication UID。
   *
   * Backend:
   * GET /members?uid=...
   */
  uid?: string;

  /** 割り当てられているBrand ID */
  brandIds?: string[];

  /** active・inactiveなど */
  status?: string;

  /** 作成日時範囲 */
  createdFrom?: string;
  createdTo?: string;

  /** 更新日時範囲 */
  updatedFrom?: string;
  updatedTo?: string;

  /** 権限名 */
  permissions?: string[];
}

/**
 * SortOrder
 * BackendのSortOrderに対応する。
 */
export type MemberSortOrder = "asc" | "desc";

/**
 * MemberSort
 * BackendのSortに対応する。
 */
export interface MemberSort {
  column: string;
  order: MemberSortOrder;
}

/**
 * MemberRepository
 *
 * IMPORTANT:
 * - Member.idはFirestoreのMemberドキュメントID
 * - Member.uidはFirebase Authentication UID
 * - GET /members/{uid}はFirebase UID専用
 * - PATCH /members/{docId}はFirestoreドキュメントID専用
 * - 一覧取得のcompanyIdはBackend側で認証中Memberから決定する
 *
 * 後方互換用のgetById・existsは廃止済み。
 */
export interface MemberRepository {
  // ===== 取得 =====

  /**
   * Firebase UIDでMemberを取得する。
   *
   * Backend:
   * GET /members/{uid}
   */
  getByUid(uid: string): Promise<Member | null>;

  /**
   * メールアドレスからMemberを取得する。
   */
  getByEmail(email: string): Promise<Member | null>;

  // ===== 一覧 =====

  /**
   * Member一覧を取得する。
   *
   * companyIdはFrontendから指定せず、
   * Backend側で認証中MemberのcompanyIdにスコープする。
   */
  list(
    page: Page,
    filter?: MemberFilter,
  ): Promise<PageResult<Member>>;

  /**
   * カーソルベースでMember一覧を取得する。
   */
  listByCursor(
    filter: MemberFilter,
    sort: MemberSort,
    cursorPage: CursorPage,
  ): Promise<CursorPageResult<Member>>;

  // ===== 作成・更新・削除 =====

  /**
   * Memberを作成する。
   *
   * 通常のConsoleからの作成では、
   * uid・id・companyIdをリクエスト本文から送信しない。
   *
   * Backend側で招待前Memberとしてuidを空にし、
   * 認証中MemberのcompanyIdを設定する。
   */
  create(
    member: Member,
    opts?: SaveOptions,
  ): Promise<Member>;

  /**
   * FirestoreのMemberドキュメントIDで更新する。
   *
   * Backend:
   * PATCH /members/{docId}
   */
  update(
    docId: string,
    patch: MemberPatch,
    opts?: SaveOptions,
  ): Promise<Member>;

  /**
   * FirestoreのMemberドキュメントIDで削除する。
   *
   * Backend:
   * DELETE /members/{docId}
   */
  delete(docId: string): Promise<void>;

  // ===== 補助 =====

  /**
   * Firebase UIDによる存在確認。
   */
  existsByUid(uid: string): Promise<boolean>;

  /**
   * 指定条件に該当するMember件数を取得する。
   */
  count(filter: MemberFilter): Promise<number>;

  /**
   * Memberを保存する。
   *
   * 現在のBackend APIでは新規作成のみ対応する。
   */
  save(
    member: Member,
    opts?: SaveOptions,
  ): Promise<Member>;
}