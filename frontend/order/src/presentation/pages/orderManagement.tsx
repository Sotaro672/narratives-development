// frontend/order/src/pages/orderManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/order.css";
import { ORDERS } from "../../infrastructure/mockdata/order_mockdata";
import type {
  Order,
  LegacyOrderStatus,
} from "../../../../shell/src/shared/types/order";

type SortKey = "orderNumber" | "createdAt" | "transferredDate" | null;
type SortDir = "asc" | "desc" | null;

// 日付フォーマット (YYYY/MM/DD)
const formatDate = (iso: string | null | undefined): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

// LegacyOrderStatus -> 表示ラベル
const statusLabel = (s: LegacyOrderStatus): string =>
  s === "paid" ? "支払済" : "移譲完了";

export default function OrderManagementPage() {
  const navigate = useNavigate();

  // ── filters ───────────────────────────────────────────────
  const [statusFilter, setStatusFilter] = useState<LegacyOrderStatus[]>([]);

  const statusOptions = useMemo(
    () =>
      Array.from(new Set(ORDERS.map((o) => o.status))).map((v) => ({
        value: v,
        label: statusLabel(v),
      })),
    []
  );

  // ── sort ─────────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>("createdAt");
  const [direction, setDirection] = useState<SortDir>("desc");

  // ── data (filter → sort) ─────────────────────────────────
  const rows = useMemo(() => {
    let data = ORDERS.filter(
      (o) =>
        statusFilter.length === 0 ||
        statusFilter.includes(o.status as LegacyOrderStatus)
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "orderNumber") {
          const cmp = a.orderNumber.localeCompare(b.orderNumber);
          return direction === "asc" ? cmp : -cmp;
        }

        const aTime = a[activeKey];
        const bTime = b[activeKey];

        const aTs =
          aTime && !Number.isNaN(Date.parse(aTime))
            ? Date.parse(aTime)
            : null;
        const bTs =
          bTime && !Number.isNaN(Date.parse(bTime))
            ? Date.parse(bTime)
            : null;

        // null の扱い: 昇順なら null は後ろ、降順なら null は前
        if (aTs === null && bTs === null) return 0;
        if (aTs === null) return direction === "asc" ? 1 : -1;
        if (bTs === null) return direction === "asc" ? -1 : 1;

        return direction === "asc" ? aTs - bTs : bTs - aTs;
      });
    }

    return data;
  }, [statusFilter, activeKey, direction]);

  // ── headers ──────────────────────────────────────────────
  const headers: React.ReactNode[] = [
    // 注文番号（Sortable）
    <SortableTableHeader
      key="orderNumber"
      label="注文番号"
      sortKey="orderNumber"
      activeKey={activeKey}
      direction={activeKey === "orderNumber" ? direction : null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
    // ユーザーID
    "ユーザーID",
    // ListID
    "リストID",
    // アイテム数
    "アイテム数",
    // ステータス（Filterable）
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(vals) => setStatusFilter(vals as LegacyOrderStatus[])}
    />,
    // 注文日（createdAt, Sortable）
    <SortableTableHeader
      key="createdAt"
      label="注文日"
      sortKey="createdAt"
      activeKey={activeKey}
      direction={activeKey === "createdAt" ? direction : null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
    // 移譲日（transferredDate, Sortable）
    <SortableTableHeader
      key="transferredDate"
      label="移譲日"
      sortKey="transferredDate"
      activeKey={activeKey}
      direction={activeKey === "transferredDate" ? direction : null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
  ];

  // 詳細ページへ遷移
  const goDetail = (orderId: string) => {
    navigate(`/order/${encodeURIComponent(orderId)}`);
  };

  return (
    <div className="p-0">
      <List
        title="注文管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setStatusFilter([]);
          setActiveKey("createdAt");
          setDirection("desc");
          console.log("注文一覧をリセット");
        }}
      >
        {rows.map((o: Order) => (
          <tr
            key={o.id}
            onClick={() => goDetail(o.id)}
            className="is-rowlink cursor-pointer hover:bg-slate-50 transition-colors"
            tabIndex={0}
            role="button"
          >
            {/* 注文番号 */}
            <td>
              <a
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  goDetail(o.id);
                }}
                className="text-blue-600 hover:underline"
              >
                {o.orderNumber}
              </a>
            </td>

            {/* ユーザーID */}
            <td>
              <div className="font-medium">{o.userId}</div>
            </td>

            {/* ListID */}
            <td>{o.listId}</td>

            {/* アイテム数 */}
            <td>{o.items.length} 点</td>

            {/* ステータス */}
            <td>
              {o.status === "paid" ? (
                <span className="order-badge is-paid">
                  {statusLabel(o.status)}
                </span>
              ) : (
                <span className="order-badge is-transferred">
                  {statusLabel(o.status)}
                </span>
              )}
            </td>

            {/* 注文日 */}
            <td>{formatDate(o.createdAt)}</td>

            {/* 移譲日 */}
            <td>{formatDate(o.transferredDate ?? null)}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
