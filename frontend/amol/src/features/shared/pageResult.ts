// frontend/amol/src/features/shared/pageResult.ts

/**
 * API層で検証・正規化されたページング結果。
 */
export type PageResult<T> = {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};

/**
 * APIから受け取る正規化前のページングレスポンス。
 *
 * バックエンドのレスポンス項目が欠けている場合を考慮し、
 * 各プロパティを任意項目として扱う。
 */
export type PageResultResponse<T> = {
  items?: T[];
  totalCount?: number;
  totalPages?: number;
  page?: number;
  perPage?: number;
};