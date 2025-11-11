// frontend/member/src/types/common.ts

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
 * - オフセット型ページングのリクエスト指定
 * - backend: common.Page 相当
 */
export interface Page {
  /** 1ページあたり取得件数 */
  limit: number;
  /** スキップ件数（0 起点） */
  offset: number;
}

/**
 * PageResult<T>
 * - オフセット型ページングのレスポンス
 * - backend: common.PageResult<T> 相当
 */
export interface PageResult<T> {
  /** 取得結果 */
  items: T[];
  /** 条件にマッチする全件数 */
  totalCount: number;
  /** この結果を得るために指定されたページ情報 */
  page: Page;
}

/**
 * CursorPage
 * - カーソル型ページングのリクエスト指定
 * - backend: common.CursorPage 相当（のフロント表現）
 */
export interface CursorPage {
  /** 1回の取得件数（デフォルト: 10件） */
  limit: number;
  /**
   * 前回レスポンスで渡された nextCursor / prevCursor を指定
   * - 初回取得時は undefined / null
   */
  cursor?: string | null;
  /**
   * 方向
   * - "next": nextCursor を使って次ページへ進む
   * - "prev": prevCursor を使って前ページへ戻る
   */
  direction?: "next" | "prev";
}

/**
 * CursorPageResult<T>
 * - カーソル型ページングのレスポンス
 * - backend: common.CursorPageResult<T> 相当
 */
export interface CursorPageResult<T> {
  /** 取得結果 */
  items: T[];
  /** 次ページ取得用カーソル（なければ null/undefined） */
  nextCursor?: string | null;
  /** 前ページ取得用カーソル（なければ null/undefined） */
  prevCursor?: string | null;
  /** 次ページが存在するかどうか */
  hasNext: boolean;
  /** 前ページが存在するかどうか */
  hasPrev: boolean;
}

/**
 * SaveOptions
 * - Repository の save/create/update 時の挙動制御オプション
 * - backend: common.SaveOptions をフロント側で表現したもの
 *
 * 実装メモ:
 * - mode が指定されていればそれを優先し、
 *   ifExists / ifNotExists は補助的なヒントとして扱う実装でOK。
 */
export interface SaveOptions {
  /**
   * 保存モード
   * - "create": 新規作成のみ（既存IDがあればエラー）
   * - "update": 既存更新のみ（存在しなければエラー）
   * - "upsert": 存在すれば更新、なければ作成
   */
  mode?: "create" | "update" | "upsert";

  /**
   * 追加条件オプション（backend側ポリシーに合わせて使用）
   * - true の場合、その条件を満たさないとエラーにする実装などを想定
   */
  ifExists?: boolean; // 「存在する場合のみ処理を許可」
  ifNotExists?: boolean; // 「存在しない場合のみ処理を許可」

  /**
   * 楽観ロック用の更新時刻条件（ISO8601）
   * - backend の updatedAt と一致している場合のみ更新などに利用可能
   */
  expectedUpdatedAt?: string;
}

/**
 * デフォルトページ（limit: 10, offset: 0）
 * - Page オブジェクトを初期化する際に利用
 */
export const DEFAULT_PAGE: Page = {
  limit: DEFAULT_PAGE_LIMIT,
  offset: 0,
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
    limit: DEFAULT_PAGE_LIMIT,
    offset: (safePage - 1) * DEFAULT_PAGE_LIMIT,
  };
}

/**
 * totalCount から総ページ数を算出
 * - UI の Pagination コンポーネントで使用
 */
export function calcTotalPages(
  totalCount: number,
  limit: number = DEFAULT_PAGE_LIMIT
): number {
  if (!totalCount || totalCount <= 0) return 1;
  return Math.max(1, Math.ceil(totalCount / limit));
}
