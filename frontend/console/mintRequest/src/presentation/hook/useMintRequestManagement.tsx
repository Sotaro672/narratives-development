// frontend/console/mintRequest/src/presentation/hook/useMintRequestManagement.tsx

import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";

import type { InspectionStatus } from "../../domain/entity/inspections";

// ✅ 3層分離：presentation -> application/usecase
import {
  loadMintRequestManagementRows,
  type ViewRow as ManagementRow,
} from "../../application/usecase/loadMintRequestManagementRows";

// ✅ presentation VM（画面の入出力型）
import type { MintRequestManagementRowVM } from "../viewModel/mintRequestManagement.vm";

// ✅ presentation formatter
import { inspectionStatusLabel } from "../formatter/inspectionStatusLabel";
import { safeDateTimeLabelJa } from "../formatter/dateJa";

// ---------------------------
// Helpers
// ---------------------------

/**
 * Date文字列 -> timestamp
 * - 解析不能や空文字は null（= sort で常に末尾）
 * - "YYYY/MM/DD" や "YYYY/MM/DD HH:mm(:ss)" の簡易フォールバックも対応
 */
const toTs = (s: string | null | undefined): number | null => {
  const v = typeof s === "string" ? s.trim() : "";
  if (!v) return null;

  const t = Date.parse(v);
  if (!Number.isNaN(t)) return t;

  const m =
    v.match(
      /^(\d{4})\/(\d{1,2})\/(\d{1,2})(?:\s+(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?$/,
    ) ?? null;

  if (!m) return null;

  const year = Number(m[1]);
  const month = Number(m[2]);
  const day = Number(m[3]);
  const hh = Number(m[4] ?? "0");
  const mm = Number(m[5] ?? "0");
  const ss = Number(m[6] ?? "0");

  const dt = new Date(year, month - 1, day, hh, mm, ss);
  const ts = dt.getTime();
  return Number.isNaN(ts) ? null : ts;
};

// Sorting key
type SortKey = "mintedAt" | "mintQuantity" | "productionQuantity" | null;

const normalizeText = (v: string | null | undefined): string => {
  return typeof v === "string" ? v.trim() : "";
};

const asInspectionStatus = (v: string): InspectionStatus | null => {
  const s = String(v ?? "").trim();
  if (s === "inspecting" || s === "completed" || s === "notYet") {
    return s as InspectionStatus;
  }
  return null;
};

const toManagementRowVM = (r: ManagementRow): MintRequestManagementRowVM => {
  return {
    ...r,

    // ✅ mintedAt 表示は "yyyy/mm/dd hh:mm:ss" に固定（dateJa.ts の確定版を利用）
    mintedAt: r.mintedAt ? safeDateTimeLabelJa(r.mintedAt, "") : null,

    // ✅ 表示ラベル
    statusLabel: inspectionStatusLabel(r.inspectionStatus),
  };
};

export const useMintRequestManagement = () => {
  const navigate = useNavigate();

  // ---------------------------
  // データ取得（usecase に委譲）
  // ---------------------------
  const [rawRows, setRawRows] = useState<MintRequestManagementRowVM[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        const rows = await loadMintRequestManagementRows();

        if (!cancelled) {
          const vms = (rows ?? []).map(toManagementRowVM);
          setRawRows(vms);
        }
      } catch (e: any) {
        if (!cancelled) setError(e?.message ?? "Failed to fetch mint requests");
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, []);

  // ---------------------------
  // Filters
  // ---------------------------
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [productionFilter, setProductionFilter] = useState<string[]>([]);
  const [requesterFilter, setRequesterFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<InspectionStatus[]>([]);

  // Sorting（デフォルト：mintedAt DESC）
  const [sortKey, setSortKey] = useState<SortKey>("mintedAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // ---------------------------
  // Filter options
  // ---------------------------
  const tokenOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => {
      const v = normalizeText(r.tokenName ?? null);
      if (v) s.add(v);
    });
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  const productionOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => {
      const v = normalizeText(r.productName ?? null);
      if (v) s.add(v);
    });
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  const requesterOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => {
      const v = normalizeText(r.createdByName ?? null);
      if (v) s.add(v);
    });
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  const statusOptions = useMemo(() => {
    const s = new Set<InspectionStatus>();
    rawRows.forEach((r) => {
      if (r.inspectionStatus) s.add(r.inspectionStatus);
    });

    return [...s].map((v) => ({
      value: v,
      label: inspectionStatusLabel(v),
    }));
  }, [rawRows]);

  // ---------------------------
  // Filter + sort rows
  // ---------------------------
  const rows = useMemo(() => {
    let data = rawRows.filter((r) => {
      const token = normalizeText(r.tokenName ?? null);
      const product = normalizeText(r.productName ?? null);
      const requester = normalizeText(r.createdByName ?? null);

      const tokenOk =
        tokenFilter.length === 0 || (token && tokenFilter.includes(token));
      const productionOk =
        productionFilter.length === 0 ||
        (product && productionFilter.includes(product));
      const requesterOk =
        requesterFilter.length === 0 || requesterFilter.includes(requester);

      const st = (r.inspectionStatus ?? ("notYet" as any)) as InspectionStatus;
      const statusOk = statusFilter.length === 0 || statusFilter.includes(st);

      return tokenOk && productionOk && requesterOk && statusOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "mintQuantity") {
          return sortDir === "asc"
            ? (a.mintQuantity ?? 0) - (b.mintQuantity ?? 0)
            : (b.mintQuantity ?? 0) - (a.mintQuantity ?? 0);
        }

        if (sortKey === "productionQuantity") {
          return sortDir === "asc"
            ? (a.productionQuantity ?? 0) - (b.productionQuantity ?? 0)
            : (b.productionQuantity ?? 0) - (a.productionQuantity ?? 0);
        }

        // mintedAt: 未設定/不正は常に末尾（asc/desc とも）
        const av = toTs(a.mintedAt ?? null);
        const bv = toTs(b.mintedAt ?? null);

        if (av === null && bv === null) return 0;
        if (av === null) return 1;
        if (bv === null) return -1;

        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [
    rawRows,
    tokenFilter,
    productionFilter,
    requesterFilter,
    statusFilter,
    sortKey,
    sortDir,
  ]);

  // ---------------------------
  // 画面遷移
  // ---------------------------
  const goDetail = (id: string) => {
    navigate(`/mintRequest/${encodeURIComponent(id)}`);
  };

  // ---------------------------
  // テーブルヘッダ
  // ---------------------------
  const headers: React.ReactNode[] = [
    <FilterableTableHeader
      key="tokenName"
      label="トークン設計"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={setTokenFilter}
    />,
    <FilterableTableHeader
      key="productName"
      label="プロダクト名"
      options={productionOptions}
      selected={productionFilter}
      onChange={setProductionFilter}
    />,
    <SortableTableHeader
      key="mintQuantity"
      label="Mint数量"
      sortKey="mintQuantity"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="productionQuantity"
      label="生産量"
      sortKey="productionQuantity"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="status"
      label="検査ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(next: string[]) => {
        const mapped = (next ?? [])
          .map((v) => asInspectionStatus(v))
          .filter((v): v is InspectionStatus => v !== null);
        setStatusFilter(mapped);
      }}
    />,
    <FilterableTableHeader
      key="requester"
      label="リクエスト者"
      options={requesterOptions}
      selected={requesterFilter}
      onChange={setRequesterFilter}
    />,
    <SortableTableHeader
      key="mintedAt"
      label="Mint実行日時"
      sortKey="mintedAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  const onReset = () => {
    setTokenFilter([]);
    setProductionFilter([]);
    setRequesterFilter([]);
    setStatusFilter([]);
    setSortKey("mintedAt");
    setSortDir("desc");
  };

  const handleRowClick = (id: string) => goDetail(id);

  const handleRowKeyDown = (
    e: React.KeyboardEvent<HTMLTableRowElement>,
    id: string,
  ) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      goDetail(id);
    }
  };

  return {
    headers,
    rows,
    onReset,
    handleRowClick,
    handleRowKeyDown,
    loading,
    error,
  };
};
