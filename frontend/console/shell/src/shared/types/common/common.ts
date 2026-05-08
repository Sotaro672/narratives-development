/**
 * 共通ページング / カーソル / SaveOptions 型定義
 * backend/internal/domain/common 配下の型イメージに対応させたフロント側表現。
 *
 * 他モジュール（member, token, brand など）でも再利用できる前提で設計。
 */

/** 1ページあたりのデフォルト件数 */
export const DEFAULT_PAGE_LIMIT = 10;

/**
 * Page
 * - オフセットではなく「ページ番号 + 件数」で指定するページング情報
 * - backend: common.Page (Number / PerPage / TotalPages) 相当
 */
export interface Page {
  /** 現在のページ番号 (1 始まり) */
  number: number;

  /** 1ページあたり取得件数 */
  perPage: number;

  /** 総ページ数（backend の Page に合わせて追加） */
  totalPages: number;
}

/**
 * PageResult<T>
 * - ページング結果のレスポンス
 * - backend: common.PageResult<T> (Items / TotalCount / TotalPages / Page / PerPage) 相当
 */
export interface PageResult<T> {
  /** 取得結果 */
  items: T[];

  /** 条件にマッチする全件数 */
  totalCount: number;

  /** 総ページ数 */
  totalPages: number;

  /** 現在のページ番号 (1 始まり) */
  page: number;

  /** 1ページあたり取得件数 */
  perPage: number;
}

/**
 * CursorPage
 * - カーソル型ページングのリクエスト指定
 * - backend: common.CursorPage 相当（のフロント表現）
 */
export interface CursorPage {
  limit: number;
  cursor?: string | null;
  direction?: "next" | "prev";
}

/**
 * CursorPageResult<T>
 * - カーソル型ページングのレスポンス
 * - backend: common.CursorPageResult<T> 相当
 */
export interface CursorPageResult<T> {
  items: T[];
  nextCursor?: string | null;
  prevCursor?: string | null;
  hasNext: boolean;
  hasPrev: boolean;
}

/**
 * SaveOptions
 * - Repository の save/create/update 時の挙動制御オプション
 * - backend: common.SaveOptions をフロント側で表現したもの
 */
export interface SaveOptions {
  mode?: "create" | "update" | "upsert";
  ifExists?: boolean;
  ifNotExists?: boolean;
  expectedUpdatedAt?: string;
}

/** 昇順/降順 */
export type SortOrder = "asc" | "desc";

/**
 * Sort
 * - backend: common.Sort に対応
 */
export interface Sort {
  column?: string;
  order?: SortOrder;
}

/**
 * デフォルトページ（page: 1, perPage: DEFAULT_PAGE_LIMIT, totalPages: 1）
 * - Page オブジェクトを初期化する際に利用
 */
export const DEFAULT_PAGE: Page = {
  number: 1,
  perPage: DEFAULT_PAGE_LIMIT,
  totalPages: 1, // ★ 追加
};

/**
 * デフォルトカーソルページ（limit: 10）
 */
export const DEFAULT_CURSOR_PAGE: CursorPage = {
  limit: DEFAULT_PAGE_LIMIT,
  cursor: null,
  direction: "next",
};

/**
 * 現在のページ番号 (1始まり) から Page を生成
 * - Pagination UI から backend query に変換する際に使用
 */
export function createPageFromCurrent(currentPage: number): Page {
  const safePage = currentPage > 0 ? currentPage : 1;
  return {
    number: safePage,
    perPage: DEFAULT_PAGE_LIMIT,
    totalPages: 1, // ★ 初期値として 1 を付与
  };
}

/**
 * totalCount から総ページ数を算出
 * - UI の Pagination コンポーネントで使用
 */
export function calcTotalPages(
  totalCount: number,
  perPage: number = DEFAULT_PAGE_LIMIT,
): number {
  if (!totalCount || totalCount <= 0) return 1;
  return Math.max(1, Math.ceil(totalCount / perPage));
}
