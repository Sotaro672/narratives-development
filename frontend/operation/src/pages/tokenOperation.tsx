// frontend/operation/src/pages/tokenOperation.tsx
import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type TokenOperation = {
  tokenName: string;
  symbol: string;
  brand: string;
  linkedProducts: number; // 連携商品種類数
  manager: string;
  planned: number; // 計画量
  requested: number; // 申請量
  issued: number; // 発行量
  distributionRate: string; // "100.0%" のような文字列
};

const TOKEN_OPERATIONS: TokenOperation[] = [
  {
    tokenName: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "山田 太郎",
    planned: 0,
    requested: 0,
    issued: 10,
    distributionRate: "100.0%",
  },
  {
    tokenName: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 10,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 0,
    requested: 0,
    issued: 12,
    distributionRate: "100.0%",
  },
  {
    tokenName: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "高橋 健太",
    planned: 0,
    requested: 5,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    linkedProducts: 0,
    manager: "高橋 健太",
    planned: 0,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
];

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

    // 配布率 ← Sortable（% を数値化して比較）
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
