// frontend/admin/shell/src/shared/ui/Table/Table.tsx
import type { ReactNode } from "react";

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
};

export default function Table<T>({ columns, rows, getRowKey, emptyMessage = "データはありません。" }: TableProps<T>) {
  if (rows.length === 0) {
    return <div className="ui-table__empty">{emptyMessage}</div>;
  }

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
            <tr key={getRowKey(row)} className="ui-table__row">
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