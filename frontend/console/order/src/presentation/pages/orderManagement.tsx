// frontend/console/order/src/presentation/pages/orderManagement.tsx
import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  SortableTableHeader,
  FilterableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/order.css";

import { createOrderRepository } from "../../infrastructure/repostiroty";
import { safeDateLabelJa } from "../../../../shell/src/shared/util/dateJa";

type SortKey = "createdAt" | null;
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

export default function OrderManagementPage() {
  const navigate = useNavigate();

  const repo = useMemo(() => createOrderRepository(), []);

  // ── filter (Token) ────────────────────────────────────────
  // 表示要件: 「移譲済み列」を「トークン列」に変更
  // filter 値: "移譲済" | "未移譲"
  type TokenFilterValue = "移譲済" | "未移譲";
  const [tokenFilter, setTokenFilter] = useState<TokenFilterValue[]>([]);

  const tokenOptions = useMemo(
    () => [
      { value: "移譲済", label: "移譲済" },
      { value: "未移譲", label: "未移譲" },
    ],
    [],
  );

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
        orderId: String((x as any).orderId ?? ""),
        listId: String((x as any).listId ?? ""),
        inventoryId: String((x as any).inventoryId ?? ""),
        avatarId: String((x as any).avatarId ?? ""),
        createdAt: String((x as any).createdAt ?? ""),
        transferred: Boolean((x as any).transferred),
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

  // ── data (filter → sort) ──────────────────────────────────
  const rows = useMemo(() => {
    // 1) filter
    let data = rowsRaw;

    if (tokenFilter.length > 0) {
      data = data.filter((r) => {
        const label: TokenFilterValue = r.transferred ? "移譲済" : "未移譲";
        return tokenFilter.includes(label);
      });
    }

    // 2) sort
    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
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
  }, [rowsRaw, tokenFilter, activeKey, direction]);

  // ── headers ──────────────────────────────────────────────
  // ✅ 注文ID列への SortableTableHeader は廃止（ただの文字列にする）
  const headers: React.ReactNode[] = [
    "注文ID",
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
    <FilterableTableHeader
      key="token"
      label="トークン"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={(vals) => setTokenFilter(vals as TokenFilterValue[])}
      dialogTitle="トークンで絞り込み"
    />,
  ];

  // 詳細ページへ遷移
  const goDetail = (id: string) => {
    navigate(`/order/${encodeURIComponent(id)}`);
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
          setTokenFilter([]);
          setActiveKey("createdAt");
          setDirection("desc");
          void fetchRows();
        }}
        showCancelButton
        onCancel={() => {
          setTokenFilter([]);
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
              <td>{safeDateLabelJa(o.createdAt, "-")}</td>
              <td>{o.transferred ? "移譲済" : "未移譲"}</td>
            </tr>
          ))
        )}
      </List>
    </div>
  );
}
