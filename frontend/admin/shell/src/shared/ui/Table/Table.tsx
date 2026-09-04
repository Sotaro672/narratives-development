// frontend/admin/shell/src/shared/ui/Table/Table.tsx
import type { KeyboardEvent, MouseEvent, ReactNode } from "react";

import "./Table.css";

export type TableColumn<T> = {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  width?: string;
  minWidth?: string;
  align?: "left" | "center" | "right";
  nowrap?: boolean;
};

type TableProps<T> = {
  columns: TableColumn<T>[];
  rows: T[];
  getRowKey: (row: T) => string;
  emptyMessage?: ReactNode;
  onRowClick?: (row: T) => void;
};

export default function Table<T>({
  columns,
  rows,
  getRowKey,
  emptyMessage = "データはありません。",
  onRowClick,
}: TableProps<T>) {
  if (rows.length === 0) {
    return <div className="ui-table__empty">{emptyMessage}</div>;
  }

  const handleRowClick = (event: MouseEvent<HTMLTableRowElement>, row: T) => {
    if (!onRowClick || isInteractiveTarget(event.target)) {
      return;
    }
    onRowClick(row);
  };

  const handleRowKeyDown = (event: KeyboardEvent<HTMLTableRowElement>, row: T) => {
    if (!onRowClick || isInteractiveTarget(event.target)) {
      return;
    }

    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onRowClick(row);
    }
  };

  return (
    <div className="ui-table__container">
      <table className="ui-table">
        <thead className="ui-table__head">
          <tr>
            {columns.map((column) => (
              <th
                key={column.key}
                className={[
                  "ui-table__header-cell",
                  `ui-table__cell--${column.align ?? "left"}`,
                  column.nowrap ? "ui-table__cell--nowrap" : "",
                ].filter(Boolean).join(" ")}
                style={{ width: column.width, minWidth: column.minWidth }}
              >
                {column.header}
              </th>
            ))}
          </tr>
        </thead>

        <tbody>
          {rows.map((row) => (
            <tr
              key={getRowKey(row)}
              className={[
                "ui-table__row",
                onRowClick ? "ui-table__row--clickable" : "",
              ].filter(Boolean).join(" ")}
              tabIndex={onRowClick ? 0 : undefined}
              onClick={(event) => handleRowClick(event, row)}
              onKeyDown={(event) => handleRowKeyDown(event, row)}
            >
              {columns.map((column) => (
                <td
                  key={column.key}
                  className={[
                    "ui-table__body-cell",
                    `ui-table__cell--${column.align ?? "left"}`,
                    column.nowrap ? "ui-table__cell--nowrap" : "",
                  ].filter(Boolean).join(" ")}
                  style={{ width: column.width, minWidth: column.minWidth }}
                >
                  {column.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
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