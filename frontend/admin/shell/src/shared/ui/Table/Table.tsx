// frontend/admin/shell/src/shared/ui/Table/Table.tsx
import {
  useMemo,
  useState,
  type KeyboardEvent,
  type MouseEvent,
  type ReactNode,
} from "react";

import "./Table.css";

type TableSortValue = string | number | boolean | null | undefined;

export type TableFilterOption = {
  value: string;
  label: string;
};

export type TableColumn<T> = {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  width?: string;
  minWidth?: string;
  align?: "left" | "center" | "right";
  nowrap?: boolean;
  sortValue?: (row: T) => TableSortValue;
  filter?: {
    getValue: (row: T) => string;
    placeholder?: string;
    options?: TableFilterOption[];
  };
};

type TableProps<T> = {
  columns: TableColumn<T>[];
  rows: T[];
  getRowKey: (row: T) => string;
  emptyMessage?: ReactNode;
  filteredEmptyMessage?: ReactNode;
  onRowClick?: (row: T) => void;
};

type SortState = {
  key: string;
  direction: "asc" | "desc";
} | null;

export default function Table<T>({
  columns,
  rows,
  getRowKey,
  emptyMessage = "データはありません。",
  filteredEmptyMessage = "条件に一致するデータはありません。",
  onRowClick,
}: TableProps<T>) {
  const [sortState, setSortState] = useState<SortState>(null);
  const [filters, setFilters] = useState<Record<string, string>>({});
  const [openFilterKey, setOpenFilterKey] = useState<string | null>(null);

  const visibleRows = useMemo(() => {
    const filteredRows = rows.filter((row) =>
      columns.every((column) => {
        if (!column.filter) {
          return true;
        }

        const filterValue = filters[column.key]?.trim() ?? "";

        if (!filterValue) {
          return true;
        }

        const rowValue = column.filter.getValue(row);

        if (column.filter.options) {
          return rowValue === filterValue;
        }

        return normalizeText(rowValue).includes(normalizeText(filterValue));
      }),
    );

    if (!sortState) {
      return filteredRows;
    }

    const sortColumn = columns.find(
      (column) => column.key === sortState.key,
    );

    if (!sortColumn?.sortValue) {
      return filteredRows;
    }

    return [...filteredRows].sort((left, right) => {
      const result = compareSortValues(
        sortColumn.sortValue!(left),
        sortColumn.sortValue!(right),
      );

      return sortState.direction === "asc" ? result : -result;
    });
  }, [columns, filters, rows, sortState]);

  const handleSort = (column: TableColumn<T>) => {
    if (!column.sortValue) {
      return;
    }

    setSortState((current) => {
      if (!current || current.key !== column.key) {
        return { key: column.key, direction: "asc" };
      }

      if (current.direction === "asc") {
        return { key: column.key, direction: "desc" };
      }

      return null;
    });
  };

  const handleFilterChange = (key: string, value: string) => {
    setFilters((current) => {
      const next = { ...current };

      if (value) {
        next[key] = value;
      } else {
        delete next[key];
      }

      return next;
    });
  };

  const handleRowClick = (
    event: MouseEvent<HTMLTableRowElement>,
    row: T,
  ) => {
    if (!onRowClick || isInteractiveTarget(event.target)) {
      return;
    }

    onRowClick(row);
  };

  const handleRowKeyDown = (
    event: KeyboardEvent<HTMLTableRowElement>,
    row: T,
  ) => {
    if (!onRowClick || isInteractiveTarget(event.target)) {
      return;
    }

    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onRowClick(row);
    }
  };

  if (rows.length === 0) {
    return <div className="ui-table__empty">{emptyMessage}</div>;
  }

  return (
    <div className="ui-table__container">
      <table className="ui-table">
        <thead className="ui-table__head">
          <tr>
            {columns.map((column) => {
              const activeSort =
                sortState?.key === column.key
                  ? sortState.direction
                  : null;
              const filterValue = filters[column.key] ?? "";
              const filterOpen = openFilterKey === column.key;

              return (
                <th
                  key={column.key}
                  className={[
                    "ui-table__header-cell",
                    `ui-table__cell--${column.align ?? "left"}`,
                    column.nowrap ? "ui-table__cell--nowrap" : "",
                  ].filter(Boolean).join(" ")}
                  style={{
                    width: column.width,
                    minWidth: column.minWidth,
                  }}
                  aria-sort={
                    column.sortValue
                      ? activeSort === "asc"
                        ? "ascending"
                        : activeSort === "desc"
                          ? "descending"
                          : "none"
                      : undefined
                  }
                >
                  <div className="ui-table__header-main">
                    <span className="ui-table__header-label">
                      {column.header}
                    </span>

                    <div className="ui-table__header-actions">
                      {column.sortValue && (
                        <button
                          type="button"
                          className={[
                            "ui-table__header-button",
                            activeSort
                              ? "ui-table__header-button--active"
                              : "",
                          ].filter(Boolean).join(" ")}
                          aria-label={`${String(column.header)}を並び替え`}
                          onClick={() => handleSort(column)}
                        >
                          <span
                            className="ui-table__sort-icon"
                            aria-hidden="true"
                          >
                            {activeSort === "asc"
                              ? "↑"
                              : activeSort === "desc"
                                ? "↓"
                                : "↕"}
                          </span>
                        </button>
                      )}

                      {column.filter && (
                        <button
                          type="button"
                          className={[
                            "ui-table__header-button",
                            filterValue
                              ? "ui-table__header-button--active"
                              : "",
                          ].filter(Boolean).join(" ")}
                          aria-label={`${String(column.header)}を絞り込む`}
                          aria-expanded={filterOpen}
                          onClick={() =>
                            setOpenFilterKey(
                              filterOpen ? null : column.key,
                            )
                          }
                        >
                          <svg
                            width="14"
                            height="14"
                            viewBox="0 0 24 24"
                            fill="none"
                            stroke="currentColor"
                            strokeWidth="2"
                            strokeLinecap="round"
                            strokeLinejoin="round"
                            aria-hidden="true"
                          >
                            <path d="M4 5h16" />
                            <path d="M7 12h10" />
                            <path d="M10 19h4" />
                          </svg>
                        </button>
                      )}
                    </div>
                  </div>

                  {column.filter && filterOpen && (
                    <div
                      className="ui-table__filter"
                      onKeyDown={(event) => {
                        if (event.key === "Escape") {
                          setOpenFilterKey(null);
                        }
                      }}
                    >
                      {column.filter.options ? (
                        <select
                          className="ui-table__filter-field"
                          value={filterValue}
                          aria-label={`${String(column.header)}の絞り込み条件`}
                          onChange={(event) =>
                            handleFilterChange(
                              column.key,
                              event.target.value,
                            )
                          }
                        >
                          <option value="">すべて</option>
                          {column.filter.options.map((option) => (
                            <option
                              key={option.value}
                              value={option.value}
                            >
                              {option.label}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <input
                          className="ui-table__filter-field"
                          type="search"
                          value={filterValue}
                          placeholder={
                            column.filter.placeholder ?? "絞り込み"
                          }
                          aria-label={`${String(column.header)}の絞り込み条件`}
                          onChange={(event) =>
                            handleFilterChange(
                              column.key,
                              event.target.value,
                            )
                          }
                        />
                      )}

                      {filterValue && (
                        <button
                          type="button"
                          className="ui-table__filter-clear"
                          onClick={() =>
                            handleFilterChange(column.key, "")
                          }
                        >
                          クリア
                        </button>
                      )}
                    </div>
                  )}
                </th>
              );
            })}
          </tr>
        </thead>

        <tbody>
          {visibleRows.length === 0 ? (
            <tr>
              <td
                className="ui-table__filtered-empty"
                colSpan={columns.length}
              >
                {filteredEmptyMessage}
              </td>
            </tr>
          ) : (
            visibleRows.map((row) => (
              <tr
                key={getRowKey(row)}
                className={[
                  "ui-table__row",
                  onRowClick
                    ? "ui-table__row--clickable"
                    : "",
                ].filter(Boolean).join(" ")}
                tabIndex={onRowClick ? 0 : undefined}
                onClick={(event) =>
                  handleRowClick(event, row)
                }
                onKeyDown={(event) =>
                  handleRowKeyDown(event, row)
                }
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={[
                      "ui-table__body-cell",
                      `ui-table__cell--${column.align ?? "left"}`,
                      column.nowrap
                        ? "ui-table__cell--nowrap"
                        : "",
                    ].filter(Boolean).join(" ")}
                    style={{
                      width: column.width,
                      minWidth: column.minWidth,
                    }}
                  >
                    {column.render(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}

function normalizeText(value: string): string {
  return value.trim().toLocaleLowerCase("ja-JP");
}

function compareSortValues(
  left: TableSortValue,
  right: TableSortValue,
): number {
  if (left == null && right == null) {
    return 0;
  }

  if (left == null) {
    return 1;
  }

  if (right == null) {
    return -1;
  }

  if (typeof left === "number" && typeof right === "number") {
    return left - right;
  }

  if (typeof left === "boolean" && typeof right === "boolean") {
    return Number(left) - Number(right);
  }

  return String(left).localeCompare(String(right), "ja-JP", {
    numeric: true,
    sensitivity: "base",
  });
}

function isInteractiveTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) {
    return false;
  }

  return Boolean(
    target.closest(
      'a, button, input, select, textarea, summary, [role="button"], [role="link"]',
    ),
  );
}