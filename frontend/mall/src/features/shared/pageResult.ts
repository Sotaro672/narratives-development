// frontend/amol/src/features/shared/pageResult.ts

/**
 * APIから受け取るページング結果。
 */
export type PageResult<T> = {
  items: T[];
  totalCount: number;
  totalPages: number;
  page: number;
  perPage: number;
};