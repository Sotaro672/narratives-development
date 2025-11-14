// frontend/production/src/presentation/pages/productionManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { PRODUCTIONS } from "../../infrastructure/mockdata/mockdata";
import type {
  Production,
  ProductionStatus,
} from "../../../../shell/src/shared/types/production";

/** ソートキー */
type SortKey = "printedAt" | "createdAt" | "totalQuantity" | null;

/** 一覧表示用に totalQuantity を付与した行型 */
type ProductionRow = Production & {
  totalQuantity: number;
};

/** ISO8601 → timestamp（不正 or 未設定は 0） */
const toTs = (iso?: string | null): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

/** ISO8601 → YYYY/M/D（不正 or 未設定は "-"） */
const formatDate = (iso?: string | null): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

export default function ProductionManagement() {
  const navigate = useNavigate();

  // ===== フィルタ状態 =====
  const [blueprintFilter, setBlueprintFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<ProductionStatus[]>([]);

  // ===== ソート状態 =====
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // ===== ベース行データ（totalQuantity 付与） =====
  const baseRows: ProductionRow[] = useMemo(
    () =>
      PRODUCTIONS.map((p) => ({
        ...p,
        totalQuantity: (p.models ?? []).reduce(
          (sum, m) => sum + (m.quantity ?? 0),
          0
        ),
      })),
    []
  );

  // ===== オプション生成 =====
  const blueprintOptions = useMemo(
    () =>
      Array.from(
        new Set(baseRows.map((p) => p.productBlueprintId))
      ).map((v) => ({ value: v, label: v })),
    [baseRows]
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(new Set(baseRows.map((p) => p.assigneeId))).map((v) => ({
        value: v,
        label: v,
      })),
    [baseRows]
  );

  const statusOptions = useMemo(
    () =>
      Array.from(new Set(baseRows.map((p) => p.status))).map((v) => ({
        value: v,
        label: v,
      })),
    [baseRows]
  );

  // ===== フィルタ＋ソート適用 =====
  const rows = useMemo(() => {
    let data = baseRows.filter((p) => {
      if (
        blueprintFilter.length > 0 &&
        !blueprintFilter.includes(p.productBlueprintId)
      ) {
        return false;
      }
      if (
        assigneeFilter.length > 0 &&
        !assigneeFilter.includes(p.assigneeId)
      ) {
        return false;
      }
      if (
        statusFilter.length > 0 &&
        !statusFilter.includes(p.status)
      ) {
        return false;
      }
      return true;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "totalQuantity") {
          const av = a.totalQuantity;
          const bv = b.totalQuantity;
          return sortDir === "asc" ? av - bv : bv - av;
        }
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [
    baseRows,
    blueprintFilter,
    assigneeFilter,
    statusFilter,
    sortKey,
    sortDir,
  ]);

  // ===== ヘッダー =====
  const headers: React.ReactNode[] = [
    "ID",
    <FilterableTableHeader
      key="blueprint"
      label="商品設計ID"
      options={blueprintOptions}
      selected={blueprintFilter}
      onChange={setBlueprintFilter}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者ID"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={setAssigneeFilter}
    />,
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter as unknown as string[]}
      onChange={(values) =>
        setStatusFilter(values as ProductionStatus[])
      }
    />,
    <SortableTableHeader
      key="totalQuantity"
      label="総生産数"
      sortKey="totalQuantity"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="printedAt"
      label="印刷日"
      sortKey="printedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="商品生産"
        headerCells={headers}
        showCreateButton
        createLabel="生産計画を作成"
        showResetButton
        onCreate={() => navigate("/production/create")}
        onReset={() => {
          setBlueprintFilter([]);
          setAssigneeFilter([]);
          setStatusFilter([]);
          setSortKey(null);
          setSortDir(null);
        }}
      >
        {rows.map((p) => (
          <tr
            key={p.id}
            className="cursor-pointer hover:bg-blue-50 transition-colors"
            onClick={() => navigate(`/production/${encodeURIComponent(p.id)}`)}
          >
            <td className="text-blue-600 underline">{p.id}</td>
            <td>{p.productBlueprintId}</td>
            <td>{p.assigneeId}</td>
            <td>{p.status}</td>
            <td>{p.totalQuantity}</td>
            <td>{formatDate(p.printedAt)}</td>
            <td>{formatDate(p.createdAt)}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
