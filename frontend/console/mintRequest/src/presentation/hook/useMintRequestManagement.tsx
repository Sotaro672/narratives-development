// frontend/console/mintRequest/src/presentation/hook/useMintRequestManagement.tsx

import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import {
  fetchMintRequestRows,
  type MintRequestRow,
  type MintRequestRowStatus,
} from "../../infrastructure/api/mintRequestApi";

// 日時文字列をタイムスタンプに変換（不正 or null は -1）
const toTs = (s: string | null | undefined): number => {
  if (!s) return -1;
  const t = Date.parse(s);
  return Number.isNaN(t) ? -1 : t;
};

// ステータス表示用ラベル
const statusLabel = (s: MintRequestRowStatus): string => {
  switch (s) {
    case "minted":
      return "Mint完了";
    case "requested":
      return "リクエスト済み";
    case "planning":
    default:
      return "計画中";
  }
};

// requestedAt をソート対象から外す
type SortKey = "mintedAt" | "mintQuantity" | null;

export const useMintRequestManagement = () => {
  const navigate = useNavigate();

  // ---------------------------
  // データ取得
  // ---------------------------
  const [rawRows, setRawRows] = useState<MintRequestRow[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);
      try {
        const rows = await fetchMintRequestRows();
        if (!cancelled) {
          setRawRows(rows);
        }
      } catch (e: any) {
        if (!cancelled) {
          setError(e?.message ?? "Failed to fetch mint requests");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
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
  const [productionFilter, setProductionFilter] = useState<string[]>([]); // プロダクト名用
  const [requesterFilter, setRequesterFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<
    MintRequestRowStatus[] | string[]
  >([]);

  // Sorting（デフォルトは Mint実行日時 の降順）
  const [sortKey, setSortKey] = useState<SortKey>("mintedAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // ---------------------------
  // Filter options
  // ---------------------------

  const tokenOptions = useMemo(() => {
    const set = new Set<string>();
    rawRows.forEach((r) => {
      if (r.tokenBlueprintId) {
        set.add(r.tokenBlueprintId);
      }
    });
    return Array.from(set).map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  // プロダクト名（productName）フィルタ
  const productionOptions = useMemo(() => {
    const set = new Set<string>();
    rawRows.forEach((r) => {
      if (r.productName && r.productName.trim()) {
        set.add(r.productName.trim());
      }
    });
    return Array.from(set).map((v) => ({
      value: v,
      label: v,
    }));
  }, [rawRows]);

  const requesterOptions = useMemo(() => {
    const set = new Set<string>();
    rawRows.forEach((r) => {
      if (r.requestedBy && r.requestedBy.trim()) {
        set.add(r.requestedBy.trim());
      }
    });
    return Array.from(set).map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  const statusOptions = useMemo(() => {
    const set = new Set<MintRequestRowStatus>();
    rawRows.forEach((r) => {
      set.add(r.status);
    });
    return Array.from(set).map((v) => ({
      value: v,
      label: statusLabel(v),
    }));
  }, [rawRows]);

  // ---------------------------
  // Filter + sort rows
  // ---------------------------

  const rows: (MintRequestRow & { statusLabel: string })[] = useMemo(() => {
    let data = rawRows.filter((r) => {
      const tokenOk =
        tokenFilter.length === 0 ||
        (r.tokenBlueprintId != null && tokenFilter.includes(r.tokenBlueprintId));
      const productionOk =
        productionFilter.length === 0 ||
        (r.productName != null && productionFilter.includes(r.productName));
      const requesterOk =
        requesterFilter.length === 0 ||
        requesterFilter.includes(r.requestedBy ?? "");
      const statusOk =
        statusFilter.length === 0 ||
        statusFilter.includes(r.status as any); // 型の都合で any 化

      return tokenOk && productionOk && requesterOk && statusOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "mintQuantity") {
          return sortDir === "asc"
            ? a.mintQuantity - b.mintQuantity
            : b.mintQuantity - a.mintQuantity;
        }

        // sortKey === "mintedAt"
        const av = toTs(a.mintedAt);
        const bv = toTs(b.mintedAt);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    // 表示用ラベルをここで付与
    return data.map((r) => ({
      ...r,
      statusLabel: statusLabel(r.status),
    }));
  }, [
    rawRows,
    tokenFilter,
    productionFilter,
    requesterFilter,
    statusFilter,
    sortKey,
    sortDir,
  ]);

  // 行クリックで詳細へ遷移（id を利用）
  const goDetail = (requestId: string) => {
    navigate(`/mintRequest/${encodeURIComponent(requestId)}`);
  };

  const headers: React.ReactNode[] = [
    // ★ ミント申請ID列は削除
    <FilterableTableHeader
      key="tokenBlueprintId"
      label="トークン設計ID"
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
    // ★ Mint数量の右隣りに生産量列を追加
    "生産量",
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(next: string[]) =>
        setStatusFilter(next as MintRequestRowStatus[] | string[])
      }
    />,
    <FilterableTableHeader
      key="requester"
      label="リクエスト者"
      options={requesterOptions}
      selected={requesterFilter}
      onChange={setRequesterFilter}
    />,
    // ★ リクエスト日時列は削除、Mint実行日時のみソート可能に残す
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

  const handleRowClick = (id: string) => {
    goDetail(id);
  };

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
