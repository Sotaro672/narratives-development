// frontend/console/shell/src/shared/types/common/common.ts

/**
 * 全ドメイン共通型
 *
 * 対象:
 * - ページ番号型ページング
 * - カーソル型ページング
 * - ページングなしの一覧レスポンス
 * - ソート
 * - Repository保存オプション
 *
 * backend/internal/domain/common 配下の型を基本としつつ、
 * Console各ドメインで現在使用されている形式との互換性も保持する。
 */

/**
 * 1ページ目を表す既定値。
 */
export const DEFAULT_PAGE_NUMBER = 1;

/**
 * 1ページあたりの既定件数。
 */
export const DEFAULT_PAGE_LIMIT = 10;

/**
 * 画面表示上の既定総ページ数。
 *
 * 0件の場合でもPagination UIを1ページ目として扱うため、
 * 既定値は1とする。
 */
export const DEFAULT_TOTAL_PAGES = 1;

// =========================================================
// ページ番号型ページング
// =========================================================

/**
 * HTTP APIのクエリパラメータとして使用するページング指定。
 *
 * 使用例:
 * GET /brands?page=1&perPage=20
 *
 * API呼び出し側で未指定を許可するため、
 * pageとperPageはoptionalとする。
 */
export interface PageParams {
  /**
   * 現在のページ番号。
   *
   * 1始まり。
   */
  page?: number;

  /**
   * 1ページあたりの取得件数。
   */
  perPage?: number;
}

/**
 * Domain・Repositoryへ渡す正規のページング指定。
 *
 * backend:
 * common.Page {
 *   Number
 *   PerPage
 * }
 *
 * totalPagesはレスポンス側の値であるため、
 * PageRequestには含めない。
 */
export interface PageRequest {
  /**
   * 現在のページ番号。
   *
   * 1始まり。
   */
  number: number;

  /**
   * 1ページあたりの取得件数。
   */
  perPage: number;
}

/**
 * Pagination UIなどで保持するページ状態。
 *
 * リクエスト情報に加えて、
 * APIレスポンスから取得した総ページ数を保持する。
 */
export interface PageState
  extends PageRequest {
  /**
   * 総ページ数。
   */
  totalPages: number;
}

/**
 * 既存コード互換用のPage型。
 *
 * 現在のMemberなどではPageにtotalPagesを保持しているため、
 * PageStateの別名として残す。
 *
 * 新規Repositoryの入力型にはPageRequestを使用し、
 * 画面状態にはPageまたはPageStateを使用する。
 */
export type Page = PageState;

/**
 * ページ番号型レスポンスのメタデータ。
 */
export interface PageMeta {
  /**
   * 条件に一致する全件数。
   */
  totalCount: number;

  /**
   * 総ページ数。
   */
  totalPages: number;

  /**
   * 現在のページ番号。
   *
   * 1始まり。
   */
  page: number;

  /**
   * 1ページあたりの取得件数。
   */
  perPage: number;
}

/**
 * 全ドメイン共通のページング結果。
 *
 * 使用例:
 * - PageResult<Member>
 * - PageResult<Brand>
 * - PageResult<TokenBlueprint>
 * - PageResult<TokenBlueprintDTO>
 * - PageResult<OrderItemInventoryRowDTO>
 * - PageResult<Permission>
 */
export interface PageResult<T>
  extends PageMeta {
  /**
   * 現在のページで取得したデータ。
   */
  items: T[];
}

/**
 * ページング情報を持たない一覧レスポンス。
 *
 * Inquiryなど、APIがitemsだけを返す場合に使用する。
 *
 * 使用例:
 * ItemsResult<InquiryManagementItem>
 */
export interface ItemsResult<T> {
  items: T[];
}

/**
 * ページングあり・なしの一覧レスポンスを扱うための共通Union型。
 *
 * レスポンス仕様がPageResultかItemsResultか確定している場合は、
 * それぞれの具体的な型を直接使用する。
 */
export type ListResult<T> =
  | PageResult<T>
  | ItemsResult<T>;

// =========================================================
// カーソル型ページング
// =========================================================

/**
 * カーソルの移動方向。
 *
 * directionはフロントエンド側で双方向ページングを
 * 実装する場合に使用する。
 */
export type CursorDirection =
  | "next"
  | "prev";

/**
 * カーソル型ページングのリクエスト指定。
 *
 * backend common.CursorPageでは、
 * afterとlimitを正規項目として使用する。
 *
 * cursorとdirectionは既存フロントエンドとの
 * 互換性を維持するため残す。
 */
export interface CursorPage {
  /**
   * 1回の取得件数。
   */
  limit: number;

  /**
   * Backendへ送信する正規カーソル。
   *
   * 未指定またはnullの場合は先頭から取得する。
   */
  after?: string | null;

  /**
   * 既存フロントエンド互換用カーソル。
   *
   * 新規実装ではafterを使用する。
   */
  cursor?: string | null;

  /**
   * フロントエンド側の移動方向。
   */
  direction?: CursorDirection;
}

/**
 * Backendのcommon.CursorPageResult<T>に対応する
 * 基本カーソルページング結果。
 */
export interface CursorPageResult<T> {
  /**
   * 取得結果。
   */
  items: T[];

  /**
   * 次ページ取得用カーソル。
   *
   * 次ページが存在しない場合はnull。
   */
  nextCursor: string | null;

  /**
   * 今回の取得上限件数。
   */
  limit: number;
}

/**
 * 前後移動に対応した拡張カーソルページング結果。
 *
 * BackendやBFF側がprevCursor、hasNext、hasPrevを
 * 返すAPIで使用する。
 */
export interface BidirectionalCursorPageResult<T>
  extends CursorPageResult<T> {
  /**
   * 前ページ取得用カーソル。
   *
   * 前ページが存在しない場合はnull。
   */
  prevCursor: string | null;

  /**
   * 次ページが存在するか。
   */
  hasNext: boolean;

  /**
   * 前ページが存在するか。
   */
  hasPrev: boolean;
}

// =========================================================
// Repository保存オプション
// =========================================================

/**
 * Repositoryの保存処理に渡す共通オプション。
 *
 * Backendのcommon.SaveOptionsと、
 * 現在のFrontend各Repositoryで使用され得る
 * 保存条件の両方に対応する。
 */
export interface SaveOptions {
  /**
   * Backendの楽観ロック用バージョン。
   *
   * backend:
   * common.SaveOptions.IfMatchVersion
   */
  ifMatchVersion?: number;

  /**
   * Frontend側で保存処理の種類を明示する場合に使用する。
   */
  mode?:
    | "create"
    | "update"
    | "upsert";

  /**
   * 対象が存在する場合のみ保存する。
   */
  ifExists?: boolean;

  /**
   * 対象が存在しない場合のみ保存する。
   */
  ifNotExists?: boolean;

  /**
   * 更新日時を利用した楽観ロック。
   *
   * ISO 8601文字列を想定する。
   */
  expectedUpdatedAt?: string;
}

// =========================================================
// ソート
// =========================================================

/**
 * ソート方向。
 */
export type SortOrder =
  | "asc"
  | "desc";

/**
 * 全ドメイン共通のソート指定。
 *
 * columnの許可値は各ドメイン側で検証する。
 */
export interface Sort {
  /**
   * ソート対象のカラム名。
   */
  column?: string;

  /**
   * 昇順または降順。
   */
  order?: SortOrder;
}

// =========================================================
// 既定値
// =========================================================

/**
 * Repositoryへ渡す正規のデフォルトページ指定。
 */
export const DEFAULT_PAGE_REQUEST:
  PageRequest = {
    number:
      DEFAULT_PAGE_NUMBER,

    perPage:
      DEFAULT_PAGE_LIMIT,
  };

/**
 * Pagination UI向けのデフォルトページ状態。
 *
 * 既存のDEFAULT_PAGE利用箇所との互換性を維持する。
 */
export const DEFAULT_PAGE:
  Page = {
    number:
      DEFAULT_PAGE_NUMBER,

    perPage:
      DEFAULT_PAGE_LIMIT,

    totalPages:
      DEFAULT_TOTAL_PAGES,
  };

/**
 * デフォルトカーソルページ。
 *
 * afterを正規値としつつ、
 * 既存互換用のcursorとdirectionも初期化する。
 */
export const DEFAULT_CURSOR_PAGE:
  CursorPage = {
    limit:
      DEFAULT_PAGE_LIMIT,

    after:
      null,

    cursor:
      null,

    direction:
      "next",
  };

// =========================================================
// 内部正規化
// =========================================================

/**
 * 値を1以上の整数へ正規化する。
 */
function normalizePositiveInteger(
  value: number,
  fallback: number,
): number {
  if (
    !Number.isFinite(
      value,
    ) ||
    value <= 0
  ) {
    return fallback;
  }

  return Math.max(
    1,
    Math.floor(
      value,
    ),
  );
}

/**
 * 値を0以上の整数へ正規化する。
 */
function normalizeNonNegativeInteger(
  value: number,
  fallback: number,
): number {
  if (
    !Number.isFinite(
      value,
    ) ||
    value < 0
  ) {
    return fallback;
  }

  return Math.max(
    0,
    Math.floor(
      value,
    ),
  );
}

// =========================================================
// ページ番号型ページング Utility
// =========================================================

/**
 * 現在のページ番号と取得件数から、
 * Repositoryへ渡すPageRequestを生成する。
 */
export function createPageRequest(
  currentPage: number,
  perPage: number =
    DEFAULT_PAGE_LIMIT,
): PageRequest {
  return {
    number:
      normalizePositiveInteger(
        currentPage,
        DEFAULT_PAGE_NUMBER,
      ),

    perPage:
      normalizePositiveInteger(
        perPage,
        DEFAULT_PAGE_LIMIT,
      ),
  };
}

/**
 * 現在のページ番号から、
 * Pagination UI向けのPageを生成する。
 *
 * 既存のcreatePageFromCurrent利用箇所との
 * 互換性を維持する。
 */
export function createPageFromCurrent(
  currentPage: number,
  perPage: number =
    DEFAULT_PAGE_LIMIT,
  totalPages: number =
    DEFAULT_TOTAL_PAGES,
): Page {
  return {
    number:
      normalizePositiveInteger(
        currentPage,
        DEFAULT_PAGE_NUMBER,
      ),

    perPage:
      normalizePositiveInteger(
        perPage,
        DEFAULT_PAGE_LIMIT,
      ),

    totalPages:
      normalizePositiveInteger(
        totalPages,
        DEFAULT_TOTAL_PAGES,
      ),
  };
}

/**
 * PageResultのページ情報から、
 * Pagination UI向けのPageを生成する。
 */
export function createPageFromResult<T>(
  result: PageResult<T>,
): Page {
  return {
    number:
      normalizePositiveInteger(
        result.page,
        DEFAULT_PAGE_NUMBER,
      ),

    perPage:
      normalizePositiveInteger(
        result.perPage,
        DEFAULT_PAGE_LIMIT,
      ),

    totalPages:
      normalizePositiveInteger(
        result.totalPages,
        DEFAULT_TOTAL_PAGES,
      ),
  };
}

/**
 * 空のPageResultを生成する。
 *
 * 各ドメインの初期状態や、
 * APIレスポンスが空の場合のフォールバックに使用する。
 */
export function createEmptyPageResult<T>(
  page: number =
    DEFAULT_PAGE_NUMBER,
  perPage: number =
    DEFAULT_PAGE_LIMIT,
): PageResult<T> {
  return {
    items: [],

    totalCount: 0,

    totalPages:
      DEFAULT_TOTAL_PAGES,

    page:
      normalizePositiveInteger(
        page,
        DEFAULT_PAGE_NUMBER,
      ),

    perPage:
      normalizePositiveInteger(
        perPage,
        DEFAULT_PAGE_LIMIT,
      ),
  };
}

/**
 * ページングなしの一覧結果を生成する。
 */
export function createItemsResult<T>(
  items: readonly T[] = [],
): ItemsResult<T> {
  return {
    items:
      Array.from(
        items,
      ),
  };
}

/**
 * PageResultのitemsだけを変換し、
 * ページング情報を維持する。
 *
 * DTOからDomain Entityへの変換などに使用する。
 *
 * 使用例:
 * mapPageResult(
 *   dtoResult,
 *   normalizeTokenBlueprint,
 * );
 */
export function mapPageResult<
  TSource,
  TTarget,
>(
  result: PageResult<TSource>,
  mapItem: (
    item: TSource,
    index: number,
  ) => TTarget,
): PageResult<TTarget> {
  return {
    items:
      result.items.map(
        mapItem,
      ),

    totalCount:
      result.totalCount,

    totalPages:
      result.totalPages,

    page:
      result.page,

    perPage:
      result.perPage,
  };
}

/**
 * ItemsResultのitemsを別の型へ変換する。
 */
export function mapItemsResult<
  TSource,
  TTarget,
>(
  result: ItemsResult<TSource>,
  mapItem: (
    item: TSource,
    index: number,
  ) => TTarget,
): ItemsResult<TTarget> {
  return {
    items:
      result.items.map(
        mapItem,
      ),
  };
}

/**
 * ListResultがPageResultかどうかを判定する。
 */
export function isPageResult<T>(
  result: ListResult<T>,
): result is PageResult<T> {
  return (
    "totalCount" in result &&
    "totalPages" in result &&
    "page" in result &&
    "perPage" in result
  );
}

/**
 * totalCountから総ページ数を算出する。
 *
 * minimumPages:
 * - Pagination UIでは1を指定する
 * - 0件時にtotalPages=0とするAPIでは0を指定できる
 */
export function calcTotalPages(
  totalCount: number,
  perPage: number =
    DEFAULT_PAGE_LIMIT,
  minimumPages: number =
    DEFAULT_TOTAL_PAGES,
): number {
  const safeTotalCount =
    normalizeNonNegativeInteger(
      totalCount,
      0,
    );

  const safePerPage =
    normalizePositiveInteger(
      perPage,
      DEFAULT_PAGE_LIMIT,
    );

  const safeMinimumPages =
    normalizeNonNegativeInteger(
      minimumPages,
      DEFAULT_TOTAL_PAGES,
    );

  if (safeTotalCount === 0) {
    return safeMinimumPages;
  }

  return Math.max(
    safeMinimumPages,
    Math.ceil(
      safeTotalCount /
        safePerPage,
    ),
  );
}

// =========================================================
// カーソル型ページング Utility
// =========================================================

/**
 * CursorPageからBackendへ送信するafterを取得する。
 *
 * afterを優先し、
 * 未設定の場合のみ既存互換用のcursorを使用する。
 */
export function resolveCursorAfter(
  page: CursorPage,
): string | null {
  return (
    page.after ??
    page.cursor ??
    null
  );
}

/**
 * カーソルページング指定を生成する。
 */
export function createCursorPage(
  after: string | null = null,
  limit: number =
    DEFAULT_PAGE_LIMIT,
): CursorPage {
  const safeLimit =
    normalizePositiveInteger(
      limit,
      DEFAULT_PAGE_LIMIT,
    );

  return {
    limit:
      safeLimit,

    after,

    cursor:
      after,

    direction:
      "next",
  };
}

/**
 * 空のCursorPageResultを生成する。
 */
export function createEmptyCursorPageResult<T>(
  limit: number =
    DEFAULT_PAGE_LIMIT,
): CursorPageResult<T> {
  return {
    items: [],

    nextCursor:
      null,

    limit:
      normalizePositiveInteger(
        limit,
        DEFAULT_PAGE_LIMIT,
      ),
  };
}

/**
 * CursorPageResultのitemsだけを変換し、
 * カーソル情報を維持する。
 */
export function mapCursorPageResult<
  TSource,
  TTarget,
>(
  result: CursorPageResult<TSource>,
  mapItem: (
    item: TSource,
    index: number,
  ) => TTarget,
): CursorPageResult<TTarget> {
  return {
    items:
      result.items.map(
        mapItem,
      ),

    nextCursor:
      result.nextCursor,

    limit:
      result.limit,
  };
}