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
 * - 本アプリケーションでは **同一 companyId のメンバーのみ** を一覧表示する運用のため、
 *   list()/count()/listByCursor() を呼ぶ際は必ず companyId を付与してください。
 *   付与を簡単にするユーティリティ: scopedFilterByCompanyId()
 */
export interface MemberFilter {
  /** 名前 / フリガナ / メール等の部分一致検索 */
  searchQuery?: string;

  /** 割当ブランドID（後方互換の Brands と同義） */
  brandIds?: string[];

  /** 所属企業IDフィルタ（※運用上は必須。ユーティリティで補完推奨） */
  companyId?: string;

  /** "active" | "inactive" など論理ステータス */
  status?: string;

  /** 作成日時範囲 (from/to) */
  createdFrom?: string; // ISO8601
  createdTo?: string;   // ISO8601

  /** 更新日時範囲 (from/to) */
  updatedFrom?: string; // ISO8601
  updatedTo?: string;   // ISO8601

  /** 権限名（Member.permissions と対応） */
  permissions?: string[];
}

/**
 * 呼び出し側で companyId を強制付与するためのユーティリティ。
 * - base に companyId が未設定でも、必ず引数 companyId を上書きします。
 * - 一覧を **同一 companyId にスコープ** させるために使用してください。
 */
export function scopedFilterByCompanyId(
  companyId: string,
  base: MemberFilter = {}
): MemberFilter {
  const id = (companyId ?? "").trim();
  if (!id) {
    throw new Error("scopedFilterByCompanyId: companyId is required");
  }
  return { ...base, companyId: id };
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
 * - 実装側は **companyId フィルタを厳密に適用** してください。
 */
export interface MemberRepository {
  // ===== 共通 CRUD / List（RepositoryCRUD, RepositoryList 相当）=====

  /** ID 取得（存在しない場合は null） */
  getById(id: string): Promise<Member | null>;

  /**
   * 一覧取得（ページング版）
   * - page: limit/offset 等を含む共通ページ情報
   * - filter: MemberFilter（※同一 companyId のみを返すようスコープ必須）
   */
  list(page: Page, filter?: MemberFilter): Promise<PageResult<Member>>;

  /**
   * 作成
   * - SaveOptions は楽観ロック / upsert など実装側で利用
   */
  create(member: Member, opts?: SaveOptions): Promise<Member>;

  /**
   * 更新（部分更新）
   * - patch の undefined は「変更なし」
   */
  update(id: string, patch: MemberPatch, opts?: SaveOptions): Promise<Member>;

  /**
   * 論理削除 or 物理削除（実装依存）
   * - backend の RepositoryCRUD.Delete 相当
   */
  delete(id: string): Promise<void>;

  // ===== 追加要件 (backend Repository 独自メソッド) =====

  /**
   * カーソルベース一覧取得
   * - filter + sort + cursorPage を指定
   * - 実装側で companyId によるスコープを必ず適用
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
   * - 実装側で companyId によるスコープを必ず適用
   */
  count(filter: MemberFilter): Promise<number>;

  /**
   * Save
   * - 新規/更新の双方を内包可能な high-level API
   * - infrastructure 側で upsert 的に扱う実装も可
   */
  save(member: Member, opts?: SaveOptions): Promise<Member>;

  /**
   * Reset
   * - 開発・テスト用。実サービス環境では no-op 実装も可。
   */
  reset(): Promise<void>;
}
