// frontend/tokenOperation/src/presentation/pages/tokenOperation.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import {
  TOKEN_OPERATION_EXTENDED,
} from "../../infrastructure/mockdata/mockdata";
import type {
  TokenOperationExtended,
} from "../../../../shell/src/shared/types/tokenOperation";

type SortKey = "tokenName" | "symbol" | "brandName" | "assigneeName" | null;
type SortDir = "asc" | "desc" | null;

export default function TokenOperationPage() {
  const navigate = useNavigate();

  // ── Filter state（ブランド・担当者） ─────────────────────────────
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ── Sort state ─────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>(null);
  const [direction, setDirection] = useState<SortDir>(null);

  // ── Filter options ─────────────────────────────────────────
  const brandOptions = useMemo(
    () =>
      Array.from(
        new Set(TOKEN_OPERATION_EXTENDED.map((r) => r.brandName)),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const assigneeOptions = useMemo(
    () =>
      Array.from(
        new Set(TOKEN_OPERATION_EXTENDED.map((r) => r.assigneeName)),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo(() => {
    let data = TOKEN_OPERATION_EXTENDED.filter(
      (r) =>
        (brandFilter.length === 0 ||
          brandFilter.includes(r.brandName)) &&
        (assigneeFilter.length === 0 ||
          assigneeFilter.includes(r.assigneeName)),
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        const av = (a[activeKey] ?? "") as string;
        const bv = (b[activeKey] ?? "") as string;
        const cmp = av.localeCompare(bv, "ja");
        return direction === "asc" ? cmp : -cmp;
      });
    }

    return data;
  }, [brandFilter, assigneeFilter, activeKey, direction]);

  // ── Table headers ─────────────────────────────────────────
  const headers: React.ReactNode[] = [
    "トークン運用ID",
    <SortableTableHeader
      key="tokenName"
      label="トークン名"
      sortKey="tokenName"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
    <SortableTableHeader
      key="symbol"
      label="シンボル"
      sortKey="symbol"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir as SortDir);
      }}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,
    <FilterableTableHeader
      key="assignee"
      label="担当者"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={setAssigneeFilter}
    />,
  ];

  // 詳細ページへ遷移（運用IDで遷移）
  const goDetail = (operationId: string) => {
    navigate(`/operation/${encodeURIComponent(operationId)}`);
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
          setAssigneeFilter([]);
          setActiveKey(null);
          setDirection(null);
        }}
      >
        {rows.map((t: TokenOperationExtended) => (
          <tr
            key={t.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(t.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(t.id);
              }
            }}
          >
            <td>{t.id}</td>
            <td>{t.tokenName}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brandName}</span>
            </td>
            <td>{t.assigneeName}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
