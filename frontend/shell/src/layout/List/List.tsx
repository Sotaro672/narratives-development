import { Children, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { Plus, RotateCw } from "lucide-react";
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

  const showPagination = totalItems > ITEMS_PER_PAGE;

  return (
    <div className="list-container">
      {/* 1段：左にタイトル、右にボタン群（同じ高さで横並び） */}
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
            <button
              className="lp-icon-btn"
              aria-label="リフレッシュ"
              title="リフレッシュ"
              onClick={onReset}
              disabled={isResetting}
            >
              <RotateCw size={18} className={isResetting ? "lp-rotate" : ""} aria-hidden />
            </button>
          )}
        </div>
      </div>

      {/* テーブル */}
      <div className="lp-card">
        <table className="lp-table" role="table" aria-label={`${title} 一覧`}>
          <thead className="lp-thead">
            <tr className="lp-row-head" role="row">
              {headerCells.map((cell, i) => (
                <th key={i} className="lp-th" scope="col" role="columnheader">
                  {cell}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="lp-tbody">{paginatedRows}</tbody>
        </table>
      </div>

      {/* ページネーション（11件以上のみ） */}
      {showPagination && (
        <div className="list-pagination">
          <button
            className="lp-btn"
            onClick={() => setPage((v) => Math.max(1, v - 1))}
            disabled={page === 1}
          >
            前へ
          </button>
          <span className="lp-page-info">
            {page} / {totalPages} ページ
          </span>
          <button
            className="lp-btn"
            onClick={() => setPage((v) => Math.min(totalPages, v + 1))}
            disabled={page === totalPages}
          >
            次へ
          </button>
        </div>
      )}
    </div>
  );
}
