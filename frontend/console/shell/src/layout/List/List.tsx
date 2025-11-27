//frontend/console/shell/src/layout/List/List.tsx
import { Children, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Plus, Trash2, X } from "lucide-react"; // ★ キャンセル(X) 追加
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

  showTrashButton?: boolean;
  onTrash?: () => void;

  // ★ 追加（キャンセルボタン）
  showCancelButton?: boolean;
  onCancel?: () => void;
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

  showTrashButton = false,
  onTrash,

  showCancelButton = false, // ★ デフォルト非表示
  onCancel,
}: ListProps) {
  // ----------------------------------------------------
  // 読み込み状態
  // ----------------------------------------------------
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(false);
  }, [children]);

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
      {/* ─────────────────────────────────────── */}
      {/* 上部ヘッダー */}
      {/* ─────────────────────────────────────── */}
      <div className="list-header one-line">
        <h1 className="list-title">{title}</h1>
        <div className="list-header-spacer" />
        <div className="list-actions-right">

          {/* ▼ 新規作成 */}
          {showCreateButton && (
            <button className="lp-btn lp-btn-primary" onClick={onCreate}>
              <Plus className="lp-btn-icon" aria-hidden />
              <span>{createLabel}</span>
            </button>
          )}

          {/* ▼ キャンセルボタン（★追加） */}
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

          {/* ▼ リフレッシュボタン（★位置を左へ移動） */}
          {showResetButton && (
            <RefreshButton
              onClick={onReset}
              loading={isResetting}
              title="リフレッシュ"
              ariaLabel="リフレッシュ"
            />
          )}

          {/* ▼ ゴミ箱ボタン（★右側→左側へ移動） */}
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

      {/* ─────────────────────────────────────── */}
      {/* テーブル領域 */}
      {/* ─────────────────────────────────────── */}
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
              {loading ? (
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
              {title} の一覧テーブル
            </TableCaption>
          </Table>
        </CardContent>
      </Card>

      {/* ─────────────────────────────────────── */}
      {/* ページネーション */}
      {/* ─────────────────────────────────────── */}
      <Pagination
        currentPage={page}
        totalPages={totalPages}
        onPageChange={setPage}
      />
    </div>
  );
}

/** 再エクスポート */
export { default as FilterableTableHeader } from "../../shared/ui/filterable-table-header";
export { default as SortableTableHeader } from "../../shared/ui/sortable-table-header";
