// frontend/console/shell/src/features/member/domain/repository/memberRepository.ts

import type { Member } from "../../../../shared/types/member";
import type {
  Page,
  PageResult,
} from "../../../../shared/types/common/common";

/**
 * MemberFilter
 * backend/internal/domain/member/repository_port.goのFilterに対応する。
 *
 * - 日付はISO 8601文字列
 * - undefinedは条件指定なし
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
 * MemberRepository
 *
 * IMPORTANT:
 * - Member.idはFirestoreのMemberドキュメントID
 * - Member.uidはFirebase Authentication UID
 * - GET /members/{uid}はFirebase UID専用
 * - 一覧取得のcompanyIdはBackend側で認証中Memberから決定する
 */
export interface MemberRepository {
  /**
   * Firebase UIDでMemberを取得する。
   *
   * Backend:
   * GET /members/{uid}
   */
  getByUid(
    uid: string,
  ): Promise<Member | null>;

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
  ): Promise<Member>;
}