// frontend/brand/src/pages/brandManagement.tsx
import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type BrandRow = {
  name: string;
  status: "active" | "inactive";
  owner: string;
  registeredAt: string; // YYYY/M/D 表示用
};

// ダミーデータ（必要に応じてAPI結果に置き換え）
const ALL_BRANDS: BrandRow[] = [
  { name: "NEXUS Street", status: "active", owner: "渡辺 花子", registeredAt: "2024/2/1" },
  { name: "LUMINA Fashion", status: "active", owner: "佐藤 美咲", registeredAt: "2024/1/1" },
];

// Utility
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "registeredAt" | null;

export default function BrandManagementPage() {
  // Filter states
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);

  const statusOptions = useMemo(
    () => Array.from(new Set(ALL_BRANDS.map((b) => b.status))).map((v) => ({
      value: v,
      label: v === "active" ? "アクティブ" : "停止",
    })),
    []
  );
  const ownerOptions = useMemo(
    () => Array.from(new Set(ALL_BRANDS.map((b) => b.owner))).map((v) => ({
      value: v,
      label: v,
    })),
    []
  );

  // Sort state
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // Data
  const rows = useMemo(() => {
    let data = ALL_BRANDS.filter(
      (b) =>
        (statusFilter.length === 0 || statusFilter.includes(b.status)) &&
        (ownerFilter.length === 0 || ownerFilter.includes(b.owner))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        return 0;
      });
    }

    return data;
  }, [statusFilter, ownerFilter, activeKey, direction]);

  // Headers
  const headers: React.ReactNode[] = [
    "ブランド名",

    // ステータス（Filterable）
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,

    // 責任者（Filterable）
    <FilterableTableHeader
      key="owner"
      label="責任者"
      options={ownerOptions}
      selected={ownerFilter}
      onChange={setOwnerFilter}
    />,

    // 登録日（Sortable）
    <SortableTableHeader
      key="registeredAt"
      label="登録日"
      sortKey="registeredAt"
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
        title="ブランド管理"
        headerCells={headers}
        showCreateButton
        createLabel="ブランド追加"
        onCreate={() => console.log("ブランド追加")}
        showResetButton
        onReset={() => {
          setStatusFilter([]);
          setOwnerFilter([]);
          setActiveKey("registeredAt");
          setDirection("desc");
          console.log("リセット");
        }}
      >
        {rows.map((b) => (
          <tr key={b.name}>
            <td>{b.name}</td>
            <td>
              <span className="lp-brand-pill">
                {b.status === "active" ? "アクティブ" : "停止"}
              </span>
            </td>
            <td>{b.owner}</td>
            <td>{b.registeredAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
