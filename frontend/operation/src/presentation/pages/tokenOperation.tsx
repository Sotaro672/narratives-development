// frontend/operation/src/pages/tokenOperation.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import {
  TOKEN_OPERATIONS,
  type TokenOperation,
} from "../../../mockdata";

// "100.0%" → 100.0
const rateToNumber = (v: string) => Number(v.replace("%", "") || 0);

type SortKey =
  | "linkedProducts"
  | "planned"
  | "requested"
  | "issued"
  | "distributionRate"
  | null;

export default function TokenOperationPage() {
  const navigate = useNavigate();

  // ── Filter state（ブランド・担当者） ─────────────────────────────
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);

  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_OPERATIONS.map((r) => r.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const managerOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_OPERATIONS.map((r) => r.manager))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ── Sort state ─────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<"asc" | "desc" | null>(null);

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(() => {
    let data = TOKEN_OPERATIONS.filter(
      (r) =>
        (brandFilter.length === 0 || brandFilter.includes(r.brand)) &&
        (managerFilter.length === 0 || managerFilter.includes(r.manager))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "distributionRate") {
          const av = rateToNumber(a.distributionRate);
          const bv = rateToNumber(b.distributionRate);
          return direction === "asc" ? av - bv : bv - av;
        }
        const av = a[activeKey] as number;
        const bv = b[activeKey] as number;
        return direction === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [brandFilter, managerFilter, activeKey, direction]);

  // ── Table headers ─────────────────────────────────────────
  const headers: React.ReactNode[] = [
    "トークン名",
    "シンボル",

    // ブランド ← Filterable
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,

    // 連携商品種類数 ← Sortable
    <SortableTableHeader
      key="linkedProducts"
      label="連携商品種類数"
      sortKey="linkedProducts"
      activeKey={activeKey ?? null}
      direction={direction ?? null}
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

    // 計画量 ← Sortable
    <SortableTableHeader
      key="planned"
      label="計画量"
      sortKey="planned"
      activeKey={activeKey ?? null}
      direction={direction ?? null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 申請量 ← Sortable
    <SortableTableHeader
      key="requested"
      label="申請量"
      sortKey="requested"
      activeKey={activeKey ?? null}
      direction={direction ?? null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 発行量 ← Sortable
    <SortableTableHeader
      key="issued"
      label="発行量"
      sortKey="issued"
      activeKey={activeKey ?? null}
      direction={direction ?? null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 配布率 ← Sortable
    <SortableTableHeader
      key="distributionRate"
      label="配布率"
      sortKey="distributionRate"
      activeKey={activeKey ?? null}
      direction={direction ?? null}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,
  ];

  // 詳細ページへ遷移（symbol を ID として使用）
  const goDetail = (symbol: string) => {
    navigate(`/operation/${encodeURIComponent(symbol)}`);
  };

  return (
    <div className="p-0">
      <List
        title="トークン運用"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setBrandFilter([]);
          setManagerFilter([]);
          setActiveKey(null);
          setDirection(null);
        }}
      >
        {rows.map((t, i) => (
          <tr
            key={i}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(t.symbol)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(t.symbol);
              }
            }}
          >
            <td>{t.tokenName}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brand}</span>
            </td>
            <td>{t.linkedProducts}</td>
            <td>{t.manager}</td>
            <td>{t.planned}</td>
            <td>{t.requested}</td>
            <td>{t.issued}</td>
            <td>{t.distributionRate}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
