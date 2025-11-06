// frontend/inventory/src/pages/inventoryManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./inventoryManagement.css";

type InventoryRow = {
  product: string;
  brand: string;
  token: string;
  total: number;
};

const INVENTORIES: InventoryRow[] = [
  {
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    total: 221,
  },
  {
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    total: 222,
  },
];

export default function InventoryManagementPage() {
  // ===== フィルタ状態 =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);

  // ヘッダー用の候補（ユニーク化）
  const productOptions = useMemo(
    () =>
      Array.from(new Set(INVENTORIES.map((r) => r.product))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(INVENTORIES.map((r) => r.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const tokenOptions = useMemo(
    () =>
      Array.from(new Set(INVENTORIES.map((r) => r.token))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ===== ソート状態（総在庫数） =====
  type SortKey = "total" | null;
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // ===== データ生成（フィルタ → ソート） =====
  const rows = useMemo(() => {
    let data = INVENTORIES.filter(
      (r) =>
        (productFilter.length === 0 || productFilter.includes(r.product)) &&
        (brandFilter.length === 0 || brandFilter.includes(r.brand)) &&
        (tokenFilter.length === 0 || tokenFilter.includes(r.token))
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) =>
        sortDir === "asc" ? a.total - b.total : b.total - a.total
      );
    }

    return data;
  }, [productFilter, brandFilter, tokenFilter, sortKey, sortDir]);

  // ===== テーブルヘッダー =====
  const headers: React.ReactNode[] = [
    <FilterableTableHeader
      key="product"
      label="プロダクト"
      options={productOptions}
      selected={productFilter}
      onChange={(vals: string[]) => setProductFilter(vals)}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={(vals: string[]) => setBrandFilter(vals)}
    />,
    <FilterableTableHeader
      key="token"
      label="トークン"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={(vals: string[]) => setTokenFilter(vals)}
    />,
    <SortableTableHeader
      key="total"
      label="総在庫数"
      sortKey="total"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0 inv-page">
      <List
        title="在庫管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setProductFilter([]);
          setBrandFilter([]);
          setTokenFilter([]);
          setSortKey(null);
          setSortDir(null);
          console.log("在庫リストを更新");
        }}
      >
        {rows.map((row, i) => (
          <tr key={`${row.product}-${row.token}-${i}`}>
            <td>{row.product}</td>
            <td>{row.brand}</td>
            <td>
              <span className="lp-brand-pill">{row.token}</span>
            </td>
            <td>
              <span className="inv__total-pill">{row.total}</span>
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
