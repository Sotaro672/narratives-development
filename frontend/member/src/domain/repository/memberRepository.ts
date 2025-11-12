// frontend/member/src/domain/repository/memberRepository.ts

import type { Member, MemberPatch } from "../entity/member";
import type {
  Page,
  PageResult,
  CursorPage,
  CursorPageResult,
  SaveOptions,
} from "../../../../shell/src/shared/types/common/common";

/**
 * MemberFilter
 * backend/internal/domain/member/repository_port.go の Filter に対応。
 *
 * - 日付は ISO8601 文字列
 * - undefined は「条件指定なし」
 */
export interface MemberFilter {
  /** 名前 / フリガナ / メール等の部分一致検索 */
  searchQuery?: string;

  /** ロールID（名称ではなくコード想定、後方互換の Roles と同義） */
  roleIds?: string[];
  /** 割当ブランドID（後方互換の Brands と同義） */
  brandIds?: string[];

  /** 所属企業IDフィルタ */
  companyId?: string;

  /** "active" | "inactive" など論理ステータス */
  status?: string;

  /** 作成日時範囲 (from/to) */
  createdFrom?: string; // ISO8601
  createdTo?: string; // ISO8601

  /** 更新日時範囲 (from/to) */
  updatedFrom?: string; // ISO8601
  updatedTo?: string; // ISO8601

  /** 権限名（Member.permissions と対応） */
  permissions?: string[];

}

/**
 * SortColumn
 * backend の SortColumn に対応
 */
export type MemberSortColumn =
  | "joinedAt"
  | "permissions"
  | "assigneeCount"
  | "name"
  | "email"
  | "updatedAt";

/**
 * SortOrder
 * backend の SortOrder に対応
 */
export type MemberSortOrder = "asc" | "desc";

/**
 * MemberSort
 * backend の Sort に対応
 */
export interface MemberSort {
  column: MemberSortColumn;
  order: MemberSortOrder;
}

/**
 * MemberRepository
 * backend/internal/domain/member/repository_port.go の Repository に対応する
 * フロントエンド側ポートインターフェース。
 *
 * - context.Context は持たず、Promise ベースで表現。
 * - infrastructure 層（Firestore / GraphQL / REST など）が実装する。
 */
export interface MemberRepository {
  // ===== 共通 CRUD / List（RepositoryCRUD, RepositoryList 相当）=====

  /** ID 取得（存在しない場合は null） */
  getById(id: string): Promise<Member | null>;

  /**
   * 一覧取得（ページング版）
   * - page: limit/offset 等を含む共通ページ情報
   * - filter: MemberFilter
   */
  list(
    page: Page,
    filter?: MemberFilter
  ): Promise<PageResult<Member>>;

  /**
   * 作成
   * - SaveOptions は楽観ロック / upsert など実装側で利用
   */
  create(
    member: Member,
    opts?: SaveOptions
  ): Promise<Member>;

  /**
   * 更新（部分更新）
   * - patch の undefined は「変更なし」
   */
  update(
    id: string,
    patch: MemberPatch,
    opts?: SaveOptions
  ): Promise<Member>;

  /**
   * 論理削除 or 物理削除（実装依存）
   * - backend の RepositoryCRUD.Delete 相当
   */
  delete(id: string): Promise<void>;

  // ===== 追加要件 (backend Repository 独自メソッド) =====

  /**
   * カーソルベース一覧取得
   * - filter + sort + cursorPage を指定
   */
  listByCursor(
    filter: MemberFilter,
    sort: MemberSort,
    cursorPage: CursorPage
  ): Promise<CursorPageResult<Member>>;

  /**
   * Email からの取得
   * - 見つからない場合は null
   */
  getByEmail(email: string): Promise<Member | null>;

  /**
   * 存在確認
   */
  exists(id: string): Promise<boolean>;

  /**
   * 件数カウント
   */
  count(filter: MemberFilter): Promise<number>;

  /**
   * Save
   * - 新規/更新の双方を内包可能な high-level API
   * - infrastructure 側で upsert 的に扱う実装も可
   */
  save(
    member: Member,
    opts?: SaveOptions
  ): Promise<Member>;

  /**
   * Reset
   * - 開発・テスト用。実サービス環境では no-op 実装も可。
   */
  reset(): Promise<void>;
}
