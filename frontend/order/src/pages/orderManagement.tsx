// frontend/order/src/pages/orderManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./orderManagement.css";

type OrderRow = {
  id: string;
  customerName: string;
  productName: string;
  additionalItems: string;
  status: "支払済" | "移譲完了";
  paymentMethod: string;
  amount: string;          // e.g. "¥32,700"
  quantityInfo: string;    // e.g. "3点"
  purchaseLocation: string;
  orderDate: string;       // YYYY/M/D
};

const ORDERS: OrderRow[] = [
  {
    id: "ORD-2024-0002",
    customerName: "山本 由紀",
    productName: "デニムジャケット ヴィンテージ加工",
    additionalItems: "他1点",
    status: "支払済",
    paymentMethod: "デジタルウォレット",
    amount: "¥32,700",
    quantityInfo: "3点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/21",
  },
  {
    id: "ORD-2024-0001",
    customerName: "Creator Alice",
    productName: "シルクブラウス プレミアムライン",
    additionalItems: "他1点",
    status: "移譲完了",
    paymentMethod: "クレジットカード",
    amount: "¥45,900",
    quantityInfo: "2点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/20",
  },
];

// utils
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};
const yenToNumber = (v: string) => {
  const n = Number(v.replace(/[^\d.-]/g, ""));
  return Number.isNaN(n) ? 0 : n;
};

type SortKey = "id" | "amount" | "orderDate" | null;

export default function OrderManagementPage() {
  // ── filters ───────────────────────────────────────────────
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [methodFilter, setMethodFilter] = useState<string[]>([]);
  const [placeFilter, setPlaceFilter] = useState<string[]>([]);

  const statusOptions = useMemo(
    () => Array.from(new Set(ORDERS.map((o) => o.status))).map((v) => ({ value: v, label: v })),
    []
  );
  const methodOptions = useMemo(
    () => Array.from(new Set(ORDERS.map((o) => o.paymentMethod))).map((v) => ({ value: v, label: v })),
    []
  );
  const placeOptions = useMemo(
    () => Array.from(new Set(ORDERS.map((o) => o.purchaseLocation))).map((v) => ({ value: v, label: v })),
    []
  );

  // ── sort ─────────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>("orderDate");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // ── data (filter → sort) ─────────────────────────────────
  const rows = useMemo(() => {
    let data = ORDERS.filter(
      (o) =>
        (statusFilter.length === 0 || statusFilter.includes(o.status)) &&
        (methodFilter.length === 0 || methodFilter.includes(o.paymentMethod)) &&
        (placeFilter.length === 0 || placeFilter.includes(o.purchaseLocation))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "id") {
          const cmp = a.id.localeCompare(b.id);
          return direction === "asc" ? cmp : -cmp;
        }
        if (activeKey === "amount") {
          const av = yenToNumber(a.amount);
          const bv = yenToNumber(b.amount);
          return direction === "asc" ? av - bv : bv - av;
        }
        // orderDate
        const av = toTs(a.orderDate);
        const bv = toTs(b.orderDate);
        return direction === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [statusFilter, methodFilter, placeFilter, activeKey, direction]);

  // ── headers ──────────────────────────────────────────────
  const headers: React.ReactNode[] = [
    // 注文番号（Sortable）
    <SortableTableHeader
      key="id"
      label="注文番号"
      sortKey="id"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,
    "お客様",
    "商品",

    // ステータス（Filterable）
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,

    // 支払方法（Filterable）
    <FilterableTableHeader
      key="paymentMethod"
      label="支払方法"
      options={methodOptions}
      selected={methodFilter}
      onChange={setMethodFilter}
    />,

    // 金額（Sortable）
    <SortableTableHeader
      key="amount"
      label="金額"
      sortKey="amount"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 購入場所（Filterable）
    <FilterableTableHeader
      key="purchaseLocation"
      label="購入場所"
      options={placeOptions}
      selected={placeFilter}
      onChange={setPlaceFilter}
    />,

    // 注文日（Sortable）
    <SortableTableHeader
      key="orderDate"
      label="注文日"
      sortKey="orderDate"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="注文管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setStatusFilter([]);
          setMethodFilter([]);
          setPlaceFilter([]);
          setActiveKey("orderDate");
          setDirection("desc");
          console.log("注文一覧を更新");
        }}
      >
        {rows.map((o) => (
          <tr key={o.id}>
            <td>{o.id}</td>
            <td>
              <div className="font-medium">{o.customerName}</div>
            </td>
            <td>
              <div>{o.productName}</div>
              <div className="order-subtext">{o.additionalItems}</div>
            </td>
            <td>
              {o.status === "支払済" ? (
                <span className="order-badge is-paid">支払済</span>
              ) : (
                <span className="order-badge is-transferred">移譲完了</span>
              )}
            </td>
            <td>{o.paymentMethod}</td>
            <td>
              <div className="font-medium">{o.amount}</div>
              <div className="order-subtext">{o.quantityInfo}</div>
            </td>
            <td>{o.purchaseLocation}</td>
            <td>{o.orderDate}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}

