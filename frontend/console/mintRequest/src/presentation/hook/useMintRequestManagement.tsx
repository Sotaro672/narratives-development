// frontend/console/mintRequest/src/presentation/hook/useMintRequestManagement.tsx

import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import {
  fetchInspectionBatches,
  fetchMintsMapByInspectionIds,
  type MintDTO,
  type InspectionBatchDTO,
} from "../../infrastructure/api/mintRequestApi";
import type { InspectionStatus } from "../../domain/entity/inspections";

// 日時文字列 → timestamp（不正や null は -1）
const toTs = (s: string | null | undefined): number => {
  if (!s) return -1;
  const t = Date.parse(s);
  return Number.isNaN(t) ? -1 : t;
};

// 🔥 検査ステータスの表示ラベル（InspectionStatus）
const inspectionStatusLabel = (s: InspectionStatus | null | undefined): string => {
  switch (s) {
    case "inspecting":
      return "検査中";
    case "completed":
      return "検査完了";
    default:
      return "未検査";
  }
};

// mint 状態（UIバッジ色などに利用）
export type MintRequestRowStatus = "planning" | "requested" | "minted";

// Sorting key
type SortKey = "mintedAt" | "mintQuantity" | null;

// 画面に必要な最小 Row（MintDTO + InspectionBatchDTO を突合して作る）
type ViewRow = {
  id: string; // = productionId (= mint.inspectionId)
  tokenBlueprintId: string | null;

  productName: string | null;

  mintQuantity: number;        // = inspection.totalPassed
  productionQuantity: number;  // = inspection.quantity

  status: MintRequestRowStatus;      // = mint の有無・minted で判定
  inspectionStatus: InspectionStatus; // = inspection.status

  createdByName: string | null; // = mint.createdByName ?? mint.createdBy
  mintedAt: string | null;      // = mint.mintedAt

  // 既存UIが使っている想定の表示用ラベル（ここでは検査ステータス）
  statusLabel: string;
};

function deriveMintStatusFromMint(mint: MintDTO | null): MintRequestRowStatus {
  if (!mint) return "planning";
  if (mint.minted || !!mint.mintedAt) return "minted";
  return "requested";
}

export const useMintRequestManagement = () => {
  const navigate = useNavigate();

  // ---------------------------
  // データ取得
  // ---------------------------
  const [rawRows, setRawRows] = useState<ViewRow[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      setLoading(true);
      setError(null);

      try {
        // 1) inspections（MintInspectionView）を取得（productName / quantity / totalPassed / status が得られる）
        const batches: InspectionBatchDTO[] = await fetchInspectionBatches();

        const productionIds = batches
          .map((b) => String((b as any).productionId ?? "").trim())
          .filter((s) => !!s);

        // 2) mints をまとめて取得（正：mintsテーブル）
        const mintMap = await fetchMintsMapByInspectionIds(productionIds);

        // 3) 画面用 Row を組み立て
        const rows: ViewRow[] = batches.map((b) => {
          const pid = String((b as any).productionId ?? "").trim();
          const mint: MintDTO | null = pid ? (mintMap[pid] ?? null) : null;

          const st = deriveMintStatusFromMint(mint);

          const inspSt = (b.status ?? "inspecting") as InspectionStatus;

          const createdByName =
            (mint?.createdByName ?? null) ||
            (mint?.createdBy ?? null) ||
            null;

          return {
            id: pid,
            tokenBlueprintId: mint?.tokenBlueprintId ?? null,

            productName: b.productName ?? null,

            mintQuantity: b.totalPassed ?? 0,
            productionQuantity: (b as any).quantity ?? (b.inspections?.length ?? 0),

            status: st,
            inspectionStatus: inspSt,

            createdByName,
            mintedAt: mint?.mintedAt ?? null,

            statusLabel: inspectionStatusLabel(inspSt),
          };
        });

        if (!cancelled) setRawRows(rows);
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
  const [statusFilter, setStatusFilter] = useState<InspectionStatus[] | string[]>(
    [],
  );

  // Sorting（デフォルト：mintedAt DESC）
  const [sortKey, setSortKey] = useState<SortKey>("mintedAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // ---------------------------
  // Filter options
  // ---------------------------

  const tokenOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.tokenBlueprintId && s.add(r.tokenBlueprintId));
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  const productionOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.productName && s.add(r.productName.trim()));
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  // ★ requestedByName / requestedBy は完全に使わない（createdByName のみ）
  const requesterOptions = useMemo(() => {
    const s = new Set<string>();
    rawRows.forEach((r) => r.createdByName && s.add(r.createdByName.trim()));
    return [...s].map((v) => ({ value: v, label: v }));
  }, [rawRows]);

  // 検査ステータスのフィルタオプション
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
      const tokenOk =
        tokenFilter.length === 0 ||
        (r.tokenBlueprintId && tokenFilter.includes(r.tokenBlueprintId));

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

    // Sort
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
