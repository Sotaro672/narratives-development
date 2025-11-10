// frontend/inventory/src/presentation/pages/inventoryManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/inventory.css";
import {
  INVENTORIES,
  type InventoryRow,
} from "../../infrastructure/mockdata/mockdata";

type SortKey = "totalQuantity" | null;

export default function InventoryManagementPage() {
  const navigate = useNavigate();

  // ===== フィルタ状態 =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);

  // ヘッダー用の候補（ユニーク化）
  const productOptions = useMemo(
    () =>
      Array.from(
        new Set(INVENTORIES.map((r) => r.productName)),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(INVENTORIES.map((r) => r.brandName)),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const tokenOptions = useMemo(
    () =>
      Array.from(
        new Set(
          INVENTORIES.map((r) => r.tokenName).filter(
            (t): t is string => !!t,
          ),
        ),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  // ===== ソート状態（総在庫数） =====
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(
    null,
  );

  // ===== データ生成（フィルタ → ソート） =====
  const rows = useMemo(() => {
    let data = INVENTORIES.filter((r) => {
      const productOk =
        productFilter.length === 0 ||
        productFilter.includes(r.productName);
      const brandOk =
        brandFilter.length === 0 ||
        brandFilter.includes(r.brandName);
      const tokenOk =
        tokenFilter.length === 0 ||
        (r.tokenName != null &&
          tokenFilter.includes(r.tokenName));
      return productOk && brandOk && tokenOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const av = a.totalQuantity;
        const bv = b.totalQuantity;
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [
    productFilter,
    brandFilter,
    tokenFilter,
    sortKey,
    sortDir,
  ]);

  // 詳細ページへの遷移
  const handleRowClick = (row: InventoryRow) => {
    navigate(`/inventory/${encodeURIComponent(row.id)}`);
  };

  return (
    <div className="p-0 inv-page">
      <List
        title="在庫管理"
        headerCells={headers(
          productOptions,
          brandOptions,
          tokenOptions,
          {
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
          },
        )}
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
        {rows.map((row) => (
          <tr
            key={row.id}
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
            <td>{row.productName}</td>
            <td>{row.brandName}</td>
            <td>
              {row.tokenName ? (
                <span className="lp-brand-pill">
                  {row.tokenName}
                </span>
              ) : (
                "-"
              )}
            </td>
            <td>
              <span className="inv__total-pill">
                {row.totalQuantity}
              </span>
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
    sortKey: SortKey;
    sortDir: "asc" | "desc" | null;
    setSortKey: (k: SortKey) => void;
    setSortDir: (d: "asc" | "desc" | null) => void;
  },
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
      key="totalQuantity"
      label="総在庫数"
      sortKey="totalQuantity"
      activeKey={ctx.sortKey}
      direction={ctx.sortDir ?? null}
      onChange={(key, dir) => {
        ctx.setSortKey(key as SortKey);
        ctx.setSortDir(dir);
      }}
    />,
  ];
}
