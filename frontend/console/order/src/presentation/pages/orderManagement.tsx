// frontend/console/order/src/presentation/pages/orderManagement.tsx
import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/order.css";

import { createOrderRepository } from "../../infrastructure/repostiroty";

type SortKey = "orderId" | "createdAt" | null;
type SortDir = "asc" | "desc" | null;

// 画面に映す値: orderId,listId,inventoryId,avatarId,createdAt,transfered:boolean
type Row = {
  orderId: string;
  listId: string;
  inventoryId: string;
  avatarId: string;
  createdAt: string;
  transferred: boolean;
};

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

export default function OrderManagementPage() {
  const navigate = useNavigate();

  // ✅ repo生成は “infrastructure” に寄せたので、ここは純粋に利用するだけ
  const repo = useMemo(() => createOrderRepository(), []);

  // ── sort ─────────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>("createdAt");
  const [direction, setDirection] = useState<SortDir>("desc");

  // ── data fetch ────────────────────────────────────────────
  const [rowsRaw, setRowsRaw] = useState<Row[]>([]);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [isResetting, setIsResetting] = useState(false);

  const fetchRows = async () => {
    setIsResetting(true);
    setErrorMsg(null);
    try {
      const res = await repo.listItemInventoryRows({ page: 1, perPage: 200 });

      const mapped: Row[] = (res.items ?? []).map((x) => ({
        orderId: String(x.orderId ?? ""),
        listId: String(x.listId ?? ""),
        inventoryId: String(x.inventoryId ?? ""),
        avatarId: String(x.avatarId ?? ""),
        createdAt: String(x.createdAt ?? ""),
        transferred: Boolean(x.transferred),
      }));

      setRowsRaw(mapped);
    } catch (e: any) {
      setRowsRaw([]);
      setErrorMsg(e?.message ? String(e.message) : "failed_to_fetch_orders");
    } finally {
      setIsResetting(false);
    }
  };

  useEffect(() => {
    void fetchRows();
  }, []);

  // ── data (sort) ───────────────────────────────────────────
  const rows = useMemo(() => {
    let data = rowsRaw;

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "orderId") {
          const cmp = a.orderId.localeCompare(b.orderId);
          return direction === "asc" ? cmp : -cmp;
        }

        const aTime = a.createdAt;
        const bTime = b.createdAt;

        const aTs =
          aTime && !Number.isNaN(Date.parse(aTime)) ? Date.parse(aTime) : null;
        const bTs =
          bTime && !Number.isNaN(Date.parse(bTime)) ? Date.parse(bTime) : null;

        if (aTs === null && bTs === null) return 0;
        if (aTs === null) return direction === "asc" ? 1 : -1;
        if (bTs === null) return direction === "asc" ? -1 : 1;

        return direction === "asc" ? aTs - bTs : bTs - aTs;
      });
    }

    return data;
  }, [rowsRaw, activeKey, direction]);

  // ── headers ──────────────────────────────────────────────
  const headers: React.ReactNode[] = [
    <SortableTableHeader
      key="orderId"
      label="注文ID"
      sortKey="orderId"
      activeKey={activeKey}
      direction={activeKey === "orderId" ? direction : null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
    "リストID",
    "在庫ID",
    "アバターID",
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
    "移譲済み",
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
        isResetting={isResetting}
        onReset={() => {
          setActiveKey("createdAt");
          setDirection("desc");
          void fetchRows();
        }}
        showCancelButton
        onCancel={() => {
          // 例: 戻る/閉じる用途にしたいなら navigate(-1) にしてもOK
          setActiveKey("createdAt");
          setDirection("desc");
          void fetchRows();
        }}
      >
        {errorMsg ? (
          <tr>
            <td colSpan={headers.length} style={{ padding: 16 }}>
              {errorMsg}
            </td>
          </tr>
        ) : (
          rows.map((o) => (
            <tr
              key={`${o.orderId}__${o.inventoryId}__${o.listId}`}
              onClick={() => goDetail(o.orderId)}
              className="is-rowlink cursor-pointer hover:bg-slate-50 transition-colors"
              tabIndex={0}
              role="button"
            >
              <td>
                <a
                  href="#"
                  onClick={(e) => {
                    e.preventDefault();
                    goDetail(o.orderId);
                  }}
                  className="text-blue-600 hover:underline"
                >
                  {o.orderId}
                </a>
              </td>

              <td>{o.listId || "-"}</td>
              <td>{o.inventoryId || "-"}</td>
              <td>{o.avatarId || "-"}</td>
              <td>{formatDate(o.createdAt)}</td>
              <td>{o.transferred ? "true" : "false"}</td>
            </tr>
          ))
        )}
      </List>
    </div>
  );
}
