// frontend/console/mintRequest/src/presentation/hook/useMintRequestManagement.tsx
import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";

import type { InspectionStatus } from "../../domain/entity/inspections";
import {
  loadMintRequestManagementRows,
  type ViewRow,
} from "../../application/mintRequestManagementService";

// 日時文字列 → timestamp（不正や null は -1）
const toTs = (s: string | null | undefined): number => {
  if (!s) return -1;
  const t = Date.parse(s);
  return Number.isNaN(t) ? -1 : t;
};

// Sorting key
type SortKey = "mintedAt" | "mintQuantity" | null;

// 🔥 検査ステータスの表示ラベル（InspectionStatus）
const inspectionStatusLabel = (
  s: InspectionStatus | null | undefined,
): string => {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
};

export const useMintRequestManagement = () => {
  const navigate = useNavigate();

  // ---------------------------
  // データ取得（serviceに委譲）
  // ---------------------------
  const [rawRows, setRawRows] = useState<ViewRow[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      // どの画面/タイミングか追いやすいように prefix を固定
      const TAG = "[mintRequest/useMintRequestManagement]";

      try {
        console.log(`${TAG} load start`);

        const rows = await loadMintRequestManagementRows();

        // ✅ 取得結果全体
        console.log(`${TAG} load result rows(length)=`, rows?.length ?? 0);
        console.log(`${TAG} load result rows(sample[0..4])=`, (rows ?? []).slice(0, 5));

        // ✅ tokenName / mintedAt / createdByName が入っているかの簡易サマリ
        const summary = (rows ?? []).slice(0, 20).map((r) => ({
          id: (r as any)?.id,
          tokenName: (r as any)?.tokenName,
          productName: (r as any)?.productName,
          createdByName: (r as any)?.createdByName,
          mintedAt: (r as any)?.mintedAt,
          inspectionStatus: (r as any)?.inspectionStatus,
          mintQuantity: (r as any)?.mintQuantity,
        }));
        console.log(`${TAG} rows(summary[0..20])=`, summary);

        // ✅ tokenName が空の行だけ抜き出して原因切り分け
        const emptyTokenName = (rows ?? []).filter((r) => !r.tokenName);
        if (emptyTokenName.length > 0) {
          console.warn(
            `${TAG} rows with empty tokenName:`,
            emptyTokenName.slice(0, 10),
          );
        }

        if (!cancelled) {
          setRawRows(rows ?? []);
          console.log(`${TAG} setRawRows done length=`, rows?.length ?? 0);
        }
      } catch (e: any) {
        console.error(`${TAG} load failed`, e);
        if (!cancelled) setError(e?.message ?? "Failed to fetch mint requests");
      } finally {
        if (!cancelled) setLoading(false);
        console.log(`${TAG} load end`);
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, []);

  // rawRows が state に入った瞬間も追えるようにする（反映漏れ切り分け）
  useEffect(() => {
    const TAG = "[mintRequest/useMintRequestManagement]";
    console.log(`${TAG} rawRows updated length=`, rawRows.length);
    console.log(`${TAG} rawRows sample[0..4]=`, rawRows.slice(0, 5));
  }, [rawRows]);

  // ---------------------------
  // Filters
  // ---------------------------
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [productionFilter, setProductionFilter] = useState<string[]>([]);
  const [requesterFilter, setRequesterFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<InspectionStatus[] | string[]>(
    [],
  );

  // Sorting（デフォルト：mintedAt DESC）
  const [sortKey, setSortKey] = useState<SortKey>("mintedAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // フィルタ状態もログ（「フィルタで全部消えてる」検知）
  useEffect(() => {
    const TAG = "[mintRequest/useMintRequestManagement]";
    console.log(`${TAG} filters updated`, {
      tokenFilter,
      productionFilter,
      requesterFilter,
      statusFilter,
      sortKey,
      sortDir,
    });
  }, [tokenFilter, productionFilter, requesterFilter, statusFilter, sortKey, sortDir]);

  // ---------------------------
  // Filter options
  // ---------------------------

  const tokenOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.tokenName && s.add(r.tokenName.trim()));
    const opts = [...s].map((v) => ({ value: v, label: v }));

    console.log("[mintRequest/useMintRequestManagement] tokenOptions=", opts);
    return opts;
  }, [rawRows]);

  const productionOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.productName && s.add(r.productName.trim()));
    const opts = [...s].map((v) => ({ value: v, label: v }));

    console.log("[mintRequest/useMintRequestManagement] productionOptions=", opts);
    return opts;
  }, [rawRows]);

  const requesterOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.createdByName && s.add(r.createdByName.trim()));
    const opts = [...s].map((v) => ({ value: v, label: v }));

    console.log("[mintRequest/useMintRequestManagement] requesterOptions=", opts);
    return opts;
  }, [rawRows]);

  const statusOptions = useMemo(() => {
    const s = new Set<InspectionStatus>();
    rawRows.forEach((r) => {
      if (r.inspectionStatus) s.add(r.inspectionStatus);
    });

    const opts = [...s].map((v) => ({
      value: v,
      label: inspectionStatusLabel(v),
    }));

    console.log("[mintRequest/useMintRequestManagement] statusOptions=", opts);
    return opts;
  }, [rawRows]);

  // ---------------------------
  // Filter + sort rows
  // ---------------------------

  const rows = useMemo(() => {
    let data = rawRows.filter((r) => {
      const tokenOk =
        tokenFilter.length === 0 ||
        (r.tokenName && tokenFilter.includes(r.tokenName));

      const productionOk =
        productionFilter.length === 0 ||
        (r.productName && productionFilter.includes(r.productName));

      const requesterOk =
        requesterFilter.length === 0 ||
        requesterFilter.includes(r.createdByName ?? "");

      const st = r.inspectionStatus ?? "notYet"; // fallback
      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(st as any);

      return tokenOk && productionOk && requesterOk && statusOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "mintQuantity") {
          return sortDir === "asc"
            ? a.mintQuantity - b.mintQuantity
            : b.mintQuantity - a.mintQuantity;
        }

        const av = toTs(a.mintedAt);
        const bv = toTs(b.mintedAt);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    console.log("[mintRequest/useMintRequestManagement] computed rows length=", data.length);
    console.log("[mintRequest/useMintRequestManagement] computed rows sample[0..4]=", data.slice(0, 5));

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
    "生産量",
    <FilterableTableHeader
      key="status"
      label="検査ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(next: string[]) =>
        setStatusFilter(next as InspectionStatus[] | string[])
      }
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
