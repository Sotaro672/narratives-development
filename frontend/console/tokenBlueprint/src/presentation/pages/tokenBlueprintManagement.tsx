// frontend/tokenBlueprint/src/pages/tokenBlueprintManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { TOKEN_BLUEPRINTS } from "../../infrastructure/mockdata/tokenBlueprint_mockdata";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";

/** ISO8601 → timestamp（不正値は 0 扱い） */
const toTs = (iso: string): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

type SortKey = "createdAt" | null;

export default function TokenBlueprintManagementPage() {
  const navigate = useNavigate();

  // フィルタ状態（brandId / assigneeId ベース）
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // オプション（brandId / assigneeId から算出）
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.brandId))).map(
        (v) => ({
          value: v,
          label: v,
        }),
      ),
    [],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(new Set(TOKEN_BLUEPRINTS.map((r) => r.assigneeId))).map(
        (v) => ({
          value: v,
          label: v,
        }),
      ),
    [],
  );

  // フィルタ + ソート適用
  const rows: TokenBlueprint[] = useMemo(() => {
    let data = TOKEN_BLUEPRINTS.filter(
      (r) =>
        (brandFilter.length === 0 || brandFilter.includes(r.brandId)) &&
        (assigneeFilter.length === 0 ||
          assigneeFilter.includes(r.assigneeId)),
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [brandFilter, assigneeFilter, sortKey, sortDir]);

  // 行クリックで詳細へ（id を使用）
  const goDetail = (id: string) => {
    navigate(`/tokenBlueprint/${encodeURIComponent(id)}`);
  };

  const headers: React.ReactNode[] = [
    "トークン名",
    "シンボル",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={(vals: string[]) => setBrandFilter(vals)}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={(vals: string[]) => setAssigneeFilter(vals)}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計"
        headerCells={headers}
        showCreateButton
        createLabel="トークン設計を作成"
        showResetButton
        onCreate={() => navigate("/tokenBlueprint/create")}
        onReset={() => {
          setBrandFilter([]);
          setAssigneeFilter([]);
          setSortKey(null);
          setSortDir(null);
          console.log("トークン設計一覧リセット");
        }}
      >
        {rows.map((t) => (
          <tr
            key={t.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => goDetail(t.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(t.id);
              }
            }}
          >
            <td>{t.name}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brandId}</span>
            </td>
            <td>{t.assigneeId}</td>
            <td>{t.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
