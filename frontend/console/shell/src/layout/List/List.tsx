// frontend/console/shell/src/layout/List/List.tsx

import { Children, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Plus, Trash2, X } from "lucide-react";

import { Button } from "../../shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../shared/ui/card";
import Empty from "../../shared/ui/empty";
import Pagination from "../../shared/ui/pagination";
import RefreshButton from "../../shared/ui/refresh";
import {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
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
  const [page, setPage] = useState(1);

  useEffect(() => {
    setLoading(false);
  }, [children]);

  const rows = useMemo(() => Children.toArray(children), [children]);
  const totalItems = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / ITEMS_PER_PAGE));

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
              <Button variant="default" size="sm" onClick={onCreate}>
                <Plus aria-hidden="true" />
                <span>{createLabel}</span>
              </Button>
            )}

            {showCancelButton && (
              <Button variant="outline" size="icon" onClick={onCancel} title="キャンセル" aria-label="キャンセル">
                <X aria-hidden="true" />
              </Button>
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
              <Button variant="destructive" size="icon" onClick={onTrash} title="ゴミ箱" aria-label="ゴミ箱">
                <Trash2 aria-hidden="true" />
              </Button>
            )}
          </div>
        </div>
      )}

      <Card className="lp-card">
        <CardHeader className="sr-only">
          <CardTitle>{tableLabel}</CardTitle>
        </CardHeader>

        <CardContent>
          <Table aria-label={tableLabel}>
            <TableHeader className="lp-thead">
              <TableRow>
                {headerCells.map((cell, i) => (
                  <TableHead key={i} scope="col">
                    {cell}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>

            <TableBody>
              {isBusy ? (
                <TableRow>
                  <TableCell colSpan={colSpan}>
                    <Empty compact description="読み込み中..." />
                  </TableCell>
                </TableRow>
              ) : totalItems === 0 ? (
                <TableRow>
                  <TableCell colSpan={colSpan}>
                    <Empty description="現在登録されている項目はございません。" />
                  </TableCell>
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

export { default as FilterableTableHeader } from "../../shared/ui/filter-able-table-header";
export { default as SortableTableHeader } from "../../shared/ui/sortable-table-header";