// frontend/console/shell/src/layout/List/List.tsx
import { Children, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Plus, Trash2, X } from "lucide-react";
import Pagination from "../../shared/ui/pagination";
import RefreshButton from "../../shared/ui/refresh";
import { Card, CardHeader, CardTitle, CardContent } from "../../shared/ui/card";
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCaption,
} from "../../shared/ui/table";
import "./List.css";

interface ListProps {
  title?: string;
  headerCells?: ReactNode[];
  children?: ReactNode;

  showCreateButton?: boolean;
  createLabel?: string;
  onCreate?: () => void;

  showResetButton?: boolean;
  onReset?: () => void;
  isResetting?: boolean;

  showTrashButton?: boolean;
  onTrash?: () => void;

  showCancelButton?: boolean;
  onCancel?: () => void;
}

const ITEMS_PER_PAGE = 10;

export default function List({
  title = "",
  headerCells = [],
  children,

  showCreateButton = false,
  createLabel = "新規作成",
  onCreate,

  showResetButton = true,
  onReset,
  isResetting = false,

  showTrashButton = false,
  onTrash,

  showCancelButton = false,
  onCancel,
}: ListProps) {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(false);
  }, [children]);

  const rows = useMemo(() => Children.toArray(children), [children]);
  const totalItems = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / ITEMS_PER_PAGE));

  const [page, setPage] = useState(1);

  useEffect(() => {
    if (page > totalPages) {
      setPage(totalPages);
    }
  }, [totalPages, page]);

  const paginatedRows = useMemo(() => {
    if (totalItems <= ITEMS_PER_PAGE) {
      return rows;
    }

    const start = (page - 1) * ITEMS_PER_PAGE;
    return rows.slice(start, start + ITEMS_PER_PAGE);
  }, [rows, totalItems, page]);

  const colSpan = Math.max(1, headerCells.length);
  const isBusy = loading || isResetting;
  const hasTitle = Boolean(title);

  const hasHeaderActions =
    showCreateButton ||
    showCancelButton ||
    showResetButton ||
    showTrashButton;

  const shouldShowHeader = hasTitle || hasHeaderActions;

  const tableLabel = title ? `${title} 一覧` : "一覧";

  return (
    <div className="list-container">
      {shouldShowHeader && (
        <div className="list-header one-line">
          {hasTitle && <h1 className="list-title">{title}</h1>}

          <div className="list-header-spacer" />

          <div className="list-actions-right">
            {showCreateButton && (
              <button className="lp-btn lp-btn-primary" onClick={onCreate}>
                <Plus className="lp-btn-icon" aria-hidden />
                <span>{createLabel}</span>
              </button>
            )}

            {showCancelButton && (
              <button
                className="lp-btn lp-btn-secondary"
                onClick={onCancel}
                title="キャンセル"
                aria-label="キャンセル"
              >
                <X className="lp-btn-icon" aria-hidden />
              </button>
            )}

            {showResetButton && (
              <RefreshButton
                onClick={onReset}
                loading={isResetting}
                title="リフレッシュ"
                ariaLabel="リフレッシュ"
              />
            )}

            {showTrashButton && (
              <button
                className="lp-btn lp-btn-danger"
                onClick={onTrash}
                title="ゴミ箱"
                aria-label="ゴミ箱"
              >
                <Trash2 className="lp-btn-icon" aria-hidden />
              </button>
            )}
          </div>
        </div>
      )}

      <Card className="lp-card">
        <CardHeader className="sr-only">
          <CardTitle>{tableLabel}</CardTitle>
        </CardHeader>

        <CardContent>
          <Table role="table" aria-label={tableLabel} className="lp-table">
            <TableHeader className="lp-thead">
              <TableRow className="lp-row-head" role="row">
                {headerCells.map((cell, i) => (
                  <TableHead
                    key={i}
                    className="lp-th"
                    scope="col"
                    role="columnheader"
                  >
                    {cell}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>

            <TableBody className="lp-tbody">
              {isBusy ? (
                <TableRow className="lp-empty-row">
                  <td className="lp-empty-cell" colSpan={colSpan}>
                    読み込み中...
                  </td>
                </TableRow>
              ) : totalItems === 0 ? (
                <TableRow className="lp-empty-row">
                  <td className="lp-empty-cell" colSpan={colSpan}>
                    現在登録されている項目はございません。
                  </td>
                </TableRow>
              ) : (
                paginatedRows
              )}
            </TableBody>

            <TableCaption className="sr-only">
              {title ? `${title} の一覧テーブル` : "一覧テーブル"}
            </TableCaption>
          </Table>
        </CardContent>
      </Card>

      <Pagination
        currentPage={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />
    </div>
  );
}

export { default as FilterableTableHeader } from "../../shared/ui/filterable-table-header";
export { default as SortableTableHeader } from "../../shared/ui/sortable-table-header";