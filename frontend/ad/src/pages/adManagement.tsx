import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./adManagement.css";

// ─────────────────────────────────────────────────────────────
// Types & Mock
// ─────────────────────────────────────────────────────────────
type AdRow = {
  campaign: string;
  brand: string;
  owner: string;
  period: string;      // "YYYY/M/D - YYYY/M/D"
  status: "実行中";
  spendRate: string;   // "66.0%"
  spend: string;       // "¥198,000"
  budget: string;      // "¥300,000"
};

const ADS: AdRow[] = [
  {
    campaign: "NEXUS Street デニムジャケット新作",
    brand: "NEXUS Street",
    owner: "渡辺 花子",
    period: "2024/3/10 - 2024/4/10",
    status: "実行中",
    spendRate: "66.0%",
    spend: "¥198,000",
    budget: "¥300,000",
  },
  {
    campaign: "LUMINA Fashion 春コレクション",
    brand: "LUMINA Fashion",
    owner: "佐藤 美咲",
    period: "2024/3/1 - 2024/3/31",
    status: "実行中",
    spendRate: "68.4%",
    spend: "¥342,000",
    budget: "¥500,000",
  },
];

// ─────────────────────────────────────────────────────────────
// Utils
// ─────────────────────────────────────────────────────────────
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v.trim(), 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};
const periodStartTs = (period: string) => {
  const start = period.split("-")[0]?.trim() ?? "";
  return toTs(start);
};
const percentToNumber = (v: string) => {
  const n = Number(v.replace("%", ""));
  return Number.isNaN(n) ? 0 : n;
};
const yenToNumber = (v: string) => {
  const n = Number(v.replace(/[^\d.-]/g, ""));
  return Number.isNaN(n) ? 0 : n;
};

type SortKey = "period" | "spendRate" | "spend" | "budget" | null;

// ─────────────────────────────────────────────────────────────
// Page
// ─────────────────────────────────────────────────────────────
export default function AdManagementPage() {
  // Filters
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  const brandOptions = useMemo(
    () => Array.from(new Set(ADS.map((a) => a.brand))).map((v) => ({ value: v, label: v })),
    []
  );
  const ownerOptions = useMemo(
    () => Array.from(new Set(ADS.map((a) => a.owner))).map((v) => ({ value: v, label: v })),
    []
  );
  const statusOptions = useMemo(
    () => Array.from(new Set(ADS.map((a) => a.status))).map((v) => ({ value: v, label: v })),
    []
  );

  // Sort
  const [activeKey, setActiveKey] = useState<SortKey>("period");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // Data (filter -> sort)
  const rows = useMemo(() => {
    let data = ADS.filter(
      (a) =>
        (brandFilter.length === 0 || brandFilter.includes(a.brand)) &&
        (ownerFilter.length === 0 || ownerFilter.includes(a.owner)) &&
        (statusFilter.length === 0 || statusFilter.includes(a.status))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "period") {
          const av = periodStartTs(a.period);
          const bv = periodStartTs(b.period);
          return direction === "asc" ? av - bv : bv - av;
        }
        if (activeKey === "spendRate") {
          const av = percentToNumber(a.spendRate);
          const bv = percentToNumber(b.spendRate);
          return direction === "asc" ? av - bv : bv - av;
        }
        if (activeKey === "spend") {
          const av = yenToNumber(a.spend);
          const bv = yenToNumber(b.spend);
          return direction === "asc" ? av - bv : bv - av;
        }
        // budget
        const av = yenToNumber(a.budget);
        const bv = yenToNumber(b.budget);
        return direction === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [brandFilter, ownerFilter, statusFilter, activeKey, direction]);

  // Headers
  const headers: React.ReactNode[] = [
    "キャンペーン名",

    // ブランド（Filterable）
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,

    // 担当者（Filterable）
    <FilterableTableHeader
      key="owner"
      label="担当者"
      options={ownerOptions}
      selected={ownerFilter}
      onChange={setOwnerFilter}
    />,

    // 広告期間（Sortable／開始日で比較）
    <SortableTableHeader
      key="period"
      label="広告期間"
      sortKey="period"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // ステータス（Filterable）
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,

    // 消化率（Sortable）
    <SortableTableHeader
      key="spendRate"
      label="消化率"
      sortKey="spendRate"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 消化（Sortable）
    <SortableTableHeader
      key="spend"
      label="消化"
      sortKey="spend"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 予算（Sortable）
    <SortableTableHeader
      key="budget"
      label="予算"
      sortKey="budget"
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
        title="広告管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setBrandFilter([]);
          setOwnerFilter([]);
          setStatusFilter([]);
          setActiveKey("period");
          setDirection("desc");
          console.log("広告一覧を更新");
        }}
      >
        {rows.map((ad) => (
          <tr key={ad.campaign}>
            <td>{ad.campaign}</td>
            <td>
              <span className="lp-brand-pill">{ad.brand}</span>
            </td>
            <td>{ad.owner}</td>
            <td>{ad.period}</td>
            <td>
              <span className="ad-status-badge">{ad.status}</span>
            </td>
            <td>{ad.spendRate}</td>
            <td>{ad.spend}</td>
            <td>{ad.budget}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
