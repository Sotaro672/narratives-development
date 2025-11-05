// frontend/list/src/pages/listManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type ListingRow = {
  id: string;
  product: string;
  brand: string;
  token: string;
  stock: number;
  manager: string;
  status: "出品中" | "停止中";
};

const LISTINGS: ListingRow[] = [
  {
    id: "list_001",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "出品中",
  },
  {
    id: "list_002",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    stock: 222,
    manager: "佐藤 美咲",
    status: "出品中",
  },
  {
    id: "list_003",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "停止中",
  },
];

type SortKey = "id" | "stock" | null;

export default function ListManagementPage() {
  // ── Filter states ─────────────────────────────────────────
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  // options for each filter
  const productOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.product))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const tokenOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.token))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const managerOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.manager))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const statusOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.status))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ── Sort state ────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>("id");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("asc");

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(() => {
    let data = LISTINGS.filter(
      (r) =>
        (productFilter.length === 0 || productFilter.includes(r.product)) &&
        (brandFilter.length === 0 || brandFilter.includes(r.brand)) &&
        (tokenFilter.length === 0 || tokenFilter.includes(r.token)) &&
        (managerFilter.length === 0 || managerFilter.includes(r.manager)) &&
        (statusFilter.length === 0 || statusFilter.includes(r.status))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "id") {
          const cmp = a.id.localeCompare(b.id);
          return direction === "asc" ? cmp : -cmp;
        }
        // stock
        return direction === "asc" ? a.stock - b.stock : b.stock - a.stock;
        // add more numeric keys here if needed
      });
    }

    return data;
  }, [
    productFilter,
    brandFilter,
    tokenFilter,
    managerFilter,
    statusFilter,
    activeKey,
    direction,
  ]);

  // ── Headers ───────────────────────────────────────────────
  const headers: React.ReactNode[] = [
    // 出品ID ← Sortable
    <SortableTableHeader
      key="id"
      label="出品ID"
      sortKey="id"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // プロダクト ← Filterable
    <FilterableTableHeader
      key="product"
      label="プロダクト"
      options={productOptions}
      selected={productFilter}
      onChange={setProductFilter}
    />,

    // ブランド ← Filterable
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,

    // トークン ← Filterable
    <FilterableTableHeader
      key="token"
      label="トークン"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={setTokenFilter}
    />,

    // 総在庫数 ← Sortable
    <SortableTableHeader
      key="stock"
      label="総在庫数"
      sortKey="stock"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 担当者 ← Filterable
    <FilterableTableHeader
      key="manager"
      label="担当者"
      options={managerOptions}
      selected={managerFilter}
      onChange={setManagerFilter}
    />,

    // ステータス ← Filterable
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="出品管理"
        headerCells={headers}
        showCreateButton
        createLabel="出品を作成"
        showResetButton
        onReset={() => {
          setProductFilter([]);
          setBrandFilter([]);
          setTokenFilter([]);
          setManagerFilter([]);
          setStatusFilter([]);
          setActiveKey("id");
          setDirection("asc");
          console.log("出品リスト更新");
        }}
      >
        {rows.map((l) => (
          <tr key={l.id}>
            <td>{l.id}</td>
            <td>{l.product}</td>
            <td>
              <span className="lp-brand-pill">{l.brand}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{l.token}</span>
            </td>
            <td>
              <span
                style={{
                  display: "inline-block",
                  minWidth: 36,
                  background: "#0b0f1a",
                  color: "#fff",
                  fontWeight: 600,
                  textAlign: "center",
                  borderRadius: 12,
                  padding: "0.2rem 0.6rem",
                }}
              >
                {l.stock}
              </span>
            </td>
            <td>{l.manager}</td>
            <td>
              {l.status === "出品中" ? (
                <span
                  style={{
                    display: "inline-block",
                    background: "#0b0f1a",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.3rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  出品中
                </span>
              ) : (
                <span
                  style={{
                    display: "inline-block",
                    background: "#ef4444",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.3rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  停止中
                </span>
              )}
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
