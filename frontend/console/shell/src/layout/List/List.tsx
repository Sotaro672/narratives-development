import { Children, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Plus } from "lucide-react";
import Pagination from "../../shared/ui/pagination";
import RefreshButton from "../../shared/ui/refresh";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../shared/ui/card";
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
  title: string;
  headerCells?: ReactNode[];
  children?: ReactNode;
  showCreateButton?: boolean;
  createLabel?: string;
  onCreate?: () => void;
  showResetButton?: boolean;
  onReset?: () => void;
  isResetting?: boolean;
}

const ITEMS_PER_PAGE = 10;

export default function List({
  title,
  headerCells = [],
  children,
  showCreateButton = false,
  createLabel = "新規作成",
  onCreate,
  showResetButton = true,
  onReset,
  isResetting = false,
}: ListProps) {
  const rows = useMemo(() => Children.toArray(children), [children]);
  const totalItems = rows.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / ITEMS_PER_PAGE));

  const [page, setPage] = useState(1);
  useEffect(() => {
    if (page > totalPages) setPage(totalPages);
  }, [totalPages, page]);

  const paginatedRows = useMemo(() => {
    if (totalItems <= ITEMS_PER_PAGE) return rows;
    const start = (page - 1) * ITEMS_PER_PAGE;
    return rows.slice(start, start + ITEMS_PER_PAGE);
  }, [rows, totalItems, page]);

  const colSpan = Math.max(1, headerCells.length);

  return (
    <div className="list-container">
      {/* ───────────────────────────────────────
          上部ヘッダー: タイトル + ボタン群
      ─────────────────────────────────────── */}
      <div className="list-header one-line">
        <h1 className="list-title">{title}</h1>
        <div className="list-header-spacer" />
        <div className="list-actions-right">
          {showCreateButton && (
            <button className="lp-btn lp-btn-primary" onClick={onCreate}>
              <Plus className="lp-btn-icon" aria-hidden />
              <span>{createLabel}</span>
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
        </div>
      </div>

      {/* ───────────────────────────────────────
          テーブル領域 (Card + Table化)
      ─────────────────────────────────────── */}
      <Card className="lp-card">
        <CardHeader className="sr-only">
          <CardTitle>{title} 一覧</CardTitle>
        </CardHeader>
        <CardContent>
          <Table role="table" aria-label={`${title} 一覧`} className="lp-table">
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
              {totalItems === 0 ? (
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
              {title} の一覧テーブル
            </TableCaption>
          </Table>
        </CardContent>
      </Card>

      {/* ───────────────────────────────────────
          ページネーション
      ─────────────────────────────────────── */}
      <Pagination
        currentPage={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />
    </div>
  );
}

/** ← 未使用エラーが出ないように、import せずに再エクスポートのみ行う */
export { default as FilterableTableHeader } from "../../shared/ui/filterable-table-header";
export { default as SortableTableHeader } from "../../shared/ui/sortable-table-header";
