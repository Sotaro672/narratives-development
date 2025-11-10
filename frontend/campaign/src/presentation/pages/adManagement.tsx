// frontend/ad/src/pages/adManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/ad.css";
import { ADS, type AdRow } from "../../infrastructure/mockdata/mockdata";

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
  const navigate = useNavigate();

  // Filters
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  const brandOptions = useMemo(
    () =>
      Array.from(new Set(ADS.map((a) => a.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const ownerOptions = useMemo(
    () =>
      Array.from(new Set(ADS.map((a) => a.owner))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const statusOptions = useMemo(
    () =>
      Array.from(new Set(ADS.map((a) => a.status))).map((v) => ({
        value: v,
        label: v,
      })),
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
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,
    <FilterableTableHeader
      key="owner"
      label="担当者"
      options={ownerOptions}
      selected={ownerFilter}
      onChange={setOwnerFilter}
    />,
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
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,
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

  // 行クリックで詳細ページへ遷移
  const goDetail = (campaign: string) => {
    navigate(`/ad/${encodeURIComponent(campaign)}`);
  };

  return (
    <div className="p-0">
      <List
        title="広告管理"
        headerCells={headers}
        showCreateButton
        createLabel="キャンペーンを作成"
        showResetButton
        onCreate={() => navigate("/ad/create")}
        onReset={() => {
          setBrandFilter([]);
          setOwnerFilter([]);
          setStatusFilter([]);
          setActiveKey("period");
          setDirection("desc");
          console.log("広告一覧を更新");
        }}
      >
        {rows.map((ad: AdRow) => (
          <tr
            key={ad.campaign}
            className="cursor-pointer hover:bg-blue-50 transition"
            onClick={() => goDetail(ad.campaign)}
          >
            <td className="text-blue-600 underline">{ad.campaign}</td>
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
