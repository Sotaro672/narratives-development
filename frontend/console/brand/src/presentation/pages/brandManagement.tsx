// frontend/brand/src/presentation/pages/brandManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/brand.css";
import {
  ALL_BRANDS,
  toBrandRows,
  type BrandRow,
} from "../../infrastructure/mockdata/mockdata";

// Utility: "YYYY/MM/DD" → timestamp
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "registeredAt" | null;
type StatusFilterValue = "active" | "inactive";

export default function BrandManagementPage() {
  const navigate = useNavigate();

  // モックBrand → 一覧表示用Rowへ変換
  const baseRows = useMemo<BrandRow[]>(() => toBrandRows(ALL_BRANDS), []);

  // ────────────────────────────────
  // フィルタ・ソート状態
  // ────────────────────────────────
  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);
  const [ownerFilter, setOwnerFilter] = useState<string[]>([]);
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // ステータス絞り込みオプション
  const statusOptions = useMemo(
    () => {
      const values = Array.from(
        new Set<StatusFilterValue>(
          baseRows.map((b) => (b.isActive ? "active" : "inactive"))
        )
      );
      return values.map(
        (v): { value: string; label: string } => ({
          value: v,
          label: v === "active" ? "アクティブ" : "停止",
        })
      );
    },
    [baseRows]
  );

  // 責任者絞り込みオプション
  const ownerOptions = useMemo(
    () => {
      const values = Array.from(new Set(baseRows.map((b) => b.owner)));
      return values.map(
        (v): { value: string; label: string } => ({
          value: v,
          label: v,
        })
      );
    },
    [baseRows]
  );

  // ────────────────────────────────
  // データフィルタリング＋ソート
  // ────────────────────────────────
  const rows = useMemo(() => {
    let data = baseRows.filter((b: BrandRow) => {
      const statusValue: StatusFilterValue = b.isActive ? "active" : "inactive";

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(statusValue);

      const ownerOk =
        ownerFilter.length === 0 || ownerFilter.includes(b.owner);

      return statusOk && ownerOk;
    });

    if (activeKey && direction) {
      data = [...data].sort((a: BrandRow, b: BrandRow) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        return 0;
      });
    }

    return data;
  }, [baseRows, statusFilter, ownerFilter, activeKey, direction]);

  // ヘッダー定義
  const headers: React.ReactNode[] = [
    "ブランド名",
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(values) =>
        setStatusFilter(values as StatusFilterValue[])
      }
    />,
    <FilterableTableHeader
      key="owner"
      label="責任者"
      options={ownerOptions}
      selected={ownerFilter}
      onChange={setOwnerFilter}
    />,
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

  const statusBadgeClass = (isActive: boolean) =>
    `brand-status-badge ${isActive ? "is-active" : "is-inactive"}`;

  // ブランド追加ボタン押下 → /brand/create へ遷移
  const handleCreateBrand = () => {
    navigate("/brand/create");
  };

  // 行クリック → /brand/:id へ遷移
  const goDetail = (brandId: string) => {
    navigate(`/brand/${encodeURIComponent(brandId)}`);
  };

  return (
    <div className="p-0">
      <List
        title="ブランド管理"
        headerCells={headers}
        showCreateButton
        createLabel="ブランド追加"
        onCreate={handleCreateBrand}
        showResetButton
        onReset={() => {
          setStatusFilter([]);
          setOwnerFilter([]);
          setActiveKey("registeredAt");
          setDirection("desc");
          console.log("ブランドリストをリセット");
        }}
      >
        {rows.map((b: BrandRow) => (
          <tr
            key={b.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => goDetail(b.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(b.id);
              }
            }}
          >
            <td>{b.name}</td>
            <td>
              <span className={statusBadgeClass(b.isActive)}>
                {b.isActive ? "アクティブ" : "停止"}
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
