// frontend/inventory/src/pages/inventoryManagement.tsx
import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
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
  const navigate = useNavigate();

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

  // 詳細ページへの遷移用ID（ブランド:商品 でエンコード）
  const toInventoryId = (r: InventoryRow) =>
    encodeURIComponent(`${r.brand}:${r.product}`);

  // 行クリック時の遷移
  const handleRowClick = (row: InventoryRow) => {
    navigate(`/inventory/${toInventoryId(row)}`);
  };

  return (
    <div className="p-0 inv-page">
      <List
        title="在庫管理"
        headerCells={headers(productOptions, brandOptions, tokenOptions, {
          productFilter,
          brandFilter,
          tokenFilter,
          setProductFilter,
          setBrandFilter,
          setTokenFilter,
          sortKey,
          sortDir,
          setSortKey,
          setSortDir,
        })}
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
          <tr
            key={`${row.product}-${row.token}-${i}`}
            className="inv__clickable-row"
            role="button"
            tabIndex={0}
            onClick={() => handleRowClick(row)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                handleRowClick(row);
              }
            }}
          >
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

/** ヘッダー生成（見通しのため分離） */
function headers(
  productOptions: Array<{ value: string; label: string }>,
  brandOptions: Array<{ value: string; label: string }>,
  tokenOptions: Array<{ value: string; label: string }>,
  ctx: {
    productFilter: string[];
    brandFilter: string[];
    tokenFilter: string[];
    setProductFilter: (v: string[]) => void;
    setBrandFilter: (v: string[]) => void;
    setTokenFilter: (v: string[]) => void;
    sortKey: "total" | null;
    sortDir: "asc" | "desc" | null;
    setSortKey: (k: "total" | null) => void;
    setSortDir: (d: "asc" | "desc" | null) => void;
  }
): React.ReactNode[] {
  return [
    <FilterableTableHeader
      key="product"
      label="プロダクト"
      options={productOptions}
      selected={ctx.productFilter}
      onChange={(vals: string[]) => ctx.setProductFilter(vals)}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={ctx.brandFilter}
      onChange={(vals: string[]) => ctx.setBrandFilter(vals)}
    />,
    <FilterableTableHeader
      key="token"
      label="トークン"
      options={tokenOptions}
      selected={ctx.tokenFilter}
      onChange={(vals: string[]) => ctx.setTokenFilter(vals)}
    />,
    <SortableTableHeader
      key="total"
      label="総在庫数"
      sortKey="total"
      activeKey={ctx.sortKey}
      direction={ctx.sortDir ?? null}
      onChange={(key, dir) => {
        ctx.setSortKey(key as "total");
        ctx.setSortDir(dir);
      }}
    />,
  ];
}
