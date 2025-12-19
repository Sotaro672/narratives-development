// frontend/console/list/src/presentation/hook/useListManagement.tsx

import React, { useMemo, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";

import {
  LISTINGS,
  getListStatusLabel,
  type ListingRow,
} from "../../infrastructure/mockdata/mockdata";

import type { ListStatus } from "../../../../shell/src/shared/types/list";

type SortKey = "id" | "stock" | null;

export type ListManagementRowVM = {
  id: string;
  productName: string;
  brandName: string;
  tokenName: string;
  stock: number;
  assigneeName: string;

  status: ListStatus;
  statusLabel: string;

  // view-only (page keeps className only)
  statusBadgeText: string;
  statusBadgeClass: string;
};

export type UseListManagementResult = {
  vm: {
    title: string;
    headers: React.ReactNode[];
    rows: ListManagementRowVM[];
  };
  handlers: {
    onReset: () => void;
    onRowClick: (id: string) => void;
    onRowKeyDown: (e: React.KeyboardEvent, id: string) => void;
  };
};

function buildStatusBadge(status: ListStatus): { text: string; className: string } {
  if (status === "listing") {
    return { text: "出品中", className: "list-status-badge is-active" };
  }
  if (status === "suspended") {
    return { text: "停止中", className: "list-status-badge is-paused" };
  }
  // deleted
  return { text: "削除済み", className: "list-status-badge is-paused" };
}

export function useListManagement(): UseListManagementResult {
  const navigate = useNavigate();

  // ── Filter states ─────────────────────────────────────────
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]); // holds ListStatus as string

  // options for each filter
  const productOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.productName))).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const brandOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.brandName))).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const tokenOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.tokenName))).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const managerOptions = useMemo(
    () =>
      Array.from(new Set(LISTINGS.map((r) => r.assigneeName))).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const statusOptions = useMemo(
    () =>
      Array.from(new Set<ListStatus>(LISTINGS.map((r) => r.status))).map(
        (status) => ({
          value: status,
          label: getListStatusLabel(status),
        }),
      ),
    [],
  );

  // ── Sort state ────────────────────────────────────────────
  const [activeKey, setActiveKey] = useState<SortKey>("id");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("asc");

  // ── Build rows (filter → sort) ────────────────────────────
  const rows = useMemo((): ListingRow[] => {
    let data = LISTINGS.filter(
      (r) =>
        (productFilter.length === 0 || productFilter.includes(r.productName)) &&
        (brandFilter.length === 0 || brandFilter.includes(r.brandName)) &&
        (tokenFilter.length === 0 || tokenFilter.includes(r.tokenName)) &&
        (managerFilter.length === 0 || managerFilter.includes(r.assigneeName)) &&
        (statusFilter.length === 0 || statusFilter.includes(r.status)),
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "id") {
          const cmp = a.id.localeCompare(b.id);
          return direction === "asc" ? cmp : -cmp;
        }
        // stock
        return direction === "asc" ? a.stock - b.stock : b.stock - a.stock;
      });
    }

    return data;
  }, [
    productFilter,
    brandFilter,
    tokenFilter,
    managerFilter,
    statusFilter,
    activeKey,
    direction,
  ]);

  // VM rows (page should be mostly style-only)
  const vmRows = useMemo((): ListManagementRowVM[] => {
    return rows.map((r) => {
      const badge = buildStatusBadge(r.status);
      return {
        id: r.id,
        productName: r.productName,
        brandName: r.brandName,
        tokenName: r.tokenName,
        stock: r.stock,
        assigneeName: r.assigneeName,

        status: r.status,
        statusLabel: getListStatusLabel(r.status),

        statusBadgeText: badge.text,
        statusBadgeClass: badge.className,
      };
    });
  }, [rows]);

  // ── Headers ───────────────────────────────────────────────
  const headers: React.ReactNode[] = useMemo(
    () => [
      // 出品ID ← Sortable
      <SortableTableHeader
        key="id"
        label="出品ID"
        sortKey="id"
        activeKey={activeKey}
        direction={direction}
        onChange={(key, dir) => {
          setActiveKey(key as SortKey);
          setDirection(dir);
        }}
      />,

      // プロダクト ← Filterable
      <FilterableTableHeader
        key="product"
        label="プロダクト"
        options={productOptions}
        selected={productFilter}
        onChange={setProductFilter}
      />,

      // ブランド ← Filterable
      <FilterableTableHeader
        key="brand"
        label="ブランド"
        options={brandOptions}
        selected={brandFilter}
        onChange={setBrandFilter}
      />,

      // トークン ← Filterable
      <FilterableTableHeader
        key="token"
        label="トークン"
        options={tokenOptions}
        selected={tokenFilter}
        onChange={setTokenFilter}
      />,

      // 総在庫数 ← Sortable
      <SortableTableHeader
        key="stock"
        label="総在庫数"
        sortKey="stock"
        activeKey={activeKey}
        direction={direction}
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

      // ステータス ← Filterable
      <FilterableTableHeader
        key="status"
        label="ステータス"
        options={statusOptions}
        selected={statusFilter}
        onChange={setStatusFilter}
      />,
    ],
    [
      activeKey,
      direction,
      productOptions,
      brandOptions,
      tokenOptions,
      managerOptions,
      statusOptions,
      productFilter,
      brandFilter,
      tokenFilter,
      managerFilter,
      statusFilter,
    ],
  );

  // handlers
  const onRowClick = useCallback(
    (id: string) => {
      navigate(`/list/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const onRowKeyDown = useCallback(
    (e: React.KeyboardEvent, id: string) => {
      if (e.key === "Enter" || e.key === " ") {
        e.preventDefault();
        navigate(`/list/${encodeURIComponent(id)}`);
      }
    },
    [navigate],
  );

  const onReset = useCallback(() => {
    setProductFilter([]);
    setBrandFilter([]);
    setTokenFilter([]);
    setManagerFilter([]);
    setStatusFilter([]);
    setActiveKey("id");
    setDirection("asc");
  }, []);

  return {
    vm: {
      title: "出品管理",
      headers,
      rows: vmRows,
    },
    handlers: {
      onReset,
      onRowClick,
      onRowKeyDown,
    },
  };
}
