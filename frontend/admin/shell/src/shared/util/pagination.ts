// frontend/admin/shell/src/shared/util/pagination.ts

export function appendPaginationParams(
  params: URLSearchParams,
  page: number | undefined,
  perPage: number | undefined,
): void {
  appendPositiveInteger(params, "page", page);
  appendPositiveInteger(params, "perPage", perPage);
}

function appendPositiveInteger(
  params: URLSearchParams,
  key: string,
  value: number | undefined,
): void {
  if (value === undefined || !Number.isInteger(value) || value <= 0) {
    return;
  }

  params.set(key, String(value));
}