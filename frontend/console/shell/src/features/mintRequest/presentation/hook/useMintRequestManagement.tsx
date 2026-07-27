// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestManagement.tsx

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../layout/List/List";

import type { InspectionStatus } from "../../domain/inspections";

// ✅ 3層分離：presentation -> application/usecase
import {
  loadMintRequestManagementRows,
  type ViewRow as ManagementRow,
} from "../../application/usecase/loadMintRequestManagementRows";

// ✅ presentation formatter
import { inspectionStatusLabel } from "../formatter/inspectionStatusLabel";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

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
type SortKey =
  | "mintedAt"
  | "mintQuantity"
  | "productionQuantity"
  | null;

const normalizeText = (
  value: string | null | undefined,
): string => {
  return typeof value === "string"
    ? value.trim()
    : "";
};

const asInspectionStatus = (
  value: string,
): InspectionStatus | null => {
  const status = String(value ?? "").trim();

  if (
    status === "inspecting" ||
    status === "completed" ||
    status === "notYet"
  ) {
    return status as InspectionStatus;
  }

  return null;
};

/**
 * 一覧画面でのみ使用する表示用の行型。
 *
 * 削除済みのMintRequestManagementRowVMには依存せず、
 * Application層のManagementRowへ表示ラベルだけを追加する。
 */
type ManagementPresentationRow = ManagementRow & {
  statusLabel: string;
};

const toPresentationRow = (
  row: ManagementRow,
): ManagementPresentationRow => {
  // mintedのときは「ミント完了」を優先し、
  // それ以外は検品ステータスを表示する。
  const statusLabel =
    row.status === "minted"
      ? "ミント完了"
      : inspectionStatusLabel(
          row.inspectionStatus,
        );

  return {
    ...row,

    // mintedAt表示は
    // "yyyy/mm/dd hh:mm:ss"に統一する。
    mintedAt: row.mintedAt
      ? safeDateTimeLabelJa(
          row.mintedAt,
          "",
        )
      : null,

    statusLabel,
  };
};

const getErrorMessage = (
  error: unknown,
): string => {
  if (error instanceof Error) {
    return error.message;
  }

  return "Failed to fetch mint requests";
};

export const useMintRequestManagement = () => {
  const navigate = useNavigate();

  // ---------------------------
  // データ取得（usecaseに委譲）
  // ---------------------------
  const [rawRows, setRawRows] = useState<
    ManagementPresentationRow[]
  >([]);

  const [loading, setLoading] =
    useState<boolean>(false);

  const [isResetting, setIsResetting] =
    useState<boolean>(false);

  const [error, setError] =
    useState<string | null>(null);

  const fetchRows = useCallback(async () => {
    setIsResetting(true);
    setLoading(true);
    setError(null);

    try {
      const rows =
        await loadMintRequestManagementRows();

      const presentationRows =
        (rows ?? []).map(
          toPresentationRow,
        );

      setRawRows(presentationRows);
    } catch (fetchError: unknown) {
      setRawRows([]);
      setError(
        getErrorMessage(fetchError),
      );
    } finally {
      setLoading(false);
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      setIsResetting(true);
      setLoading(true);
      setError(null);

      try {
        const rows =
          await loadMintRequestManagementRows();

        if (!cancelled) {
          const presentationRows =
            (rows ?? []).map(
              toPresentationRow,
            );

          setRawRows(
            presentationRows,
          );
        }
      } catch (fetchError: unknown) {
        if (!cancelled) {
          setRawRows([]);
          setError(
            getErrorMessage(
              fetchError,
            ),
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
          setIsResetting(false);
        }
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, []);

  // ---------------------------
  // Filters
  // ---------------------------
  const [tokenFilter, setTokenFilter] =
    useState<string[]>([]);

  const [
    productionFilter,
    setProductionFilter,
  ] = useState<string[]>([]);

  const [
    requesterFilter,
    setRequesterFilter,
  ] = useState<string[]>([]);

  const [statusFilter, setStatusFilter] =
    useState<InspectionStatus[]>([]);

  // Sorting（デフォルト：mintedAt DESC）
  const [sortKey, setSortKey] =
    useState<SortKey>("mintedAt");

  const [sortDir, setSortDir] =
    useState<
      "asc" | "desc" | null
    >("desc");

  // ---------------------------
  // Filter options
  // ---------------------------
  const tokenOptions = useMemo(() => {
    const values = new Set<string>();

    rawRows.forEach((row) => {
      const value = normalizeText(
        row.tokenName,
      );

      if (value) {
        values.add(value);
      }
    });

    return [...values].map(
      (value) => ({
        value,
        label: value,
      }),
    );
  }, [rawRows]);

  const productionOptions =
    useMemo(() => {
      const values = new Set<string>();

      rawRows.forEach((row) => {
        const value = normalizeText(
          row.productName,
        );

        if (value) {
          values.add(value);
        }
      });

      return [...values].map(
        (value) => ({
          value,
          label: value,
        }),
      );
    }, [rawRows]);

  const requesterOptions =
    useMemo(() => {
      const values = new Set<string>();

      rawRows.forEach((row) => {
        const value = normalizeText(
          row.requestedByName,
        );

        if (value) {
          values.add(value);
        }
      });

      return [...values].map(
        (value) => ({
          value,
          label: value,
        }),
      );
    }, [rawRows]);

  const statusOptions = useMemo(() => {
    const values =
      new Set<InspectionStatus>();

    rawRows.forEach((row) => {
      if (row.inspectionStatus) {
        values.add(
          row.inspectionStatus,
        );
      }
    });

    return [...values].map(
      (value) => ({
        value,
        label:
          inspectionStatusLabel(
            value,
          ),
      }),
    );
  }, [rawRows]);

  // ---------------------------
  // Filter + sort rows
  // ---------------------------
  const rows = useMemo(() => {
    let data = rawRows.filter(
      (row) => {
        const token = normalizeText(
          row.tokenName,
        );

        const product =
          normalizeText(
            row.productName,
          );

        const requester =
          normalizeText(
            row.requestedByName,
          );

        const tokenOk =
          tokenFilter.length === 0 ||
          (
            token !== "" &&
            tokenFilter.includes(
              token,
            )
          );

        const productionOk =
          productionFilter.length === 0 ||
          (
            product !== "" &&
            productionFilter.includes(
              product,
            )
          );

        const requesterOk =
          requesterFilter.length === 0 ||
          requesterFilter.includes(
            requester,
          );

        const statusOk =
          statusFilter.length === 0 ||
          statusFilter.includes(
            row.inspectionStatus,
          );

        return (
          tokenOk &&
          productionOk &&
          requesterOk &&
          statusOk
        );
      },
    );

    if (sortKey && sortDir) {
      data = [...data].sort(
        (a, b) => {
          if (
            sortKey ===
            "mintQuantity"
          ) {
            return sortDir === "asc"
              ? a.mintQuantity -
                  b.mintQuantity
              : b.mintQuantity -
                  a.mintQuantity;
          }

          if (
            sortKey ===
            "productionQuantity"
          ) {
            return sortDir === "asc"
              ? a.productionQuantity -
                  b.productionQuantity
              : b.productionQuantity -
                  a.productionQuantity;
          }

          // mintedAt未設定・不正値は、
          // asc/descにかかわらず末尾にする。
          const aTimestamp = toTs(
            a.mintedAt,
          );

          const bTimestamp = toTs(
            b.mintedAt,
          );

          if (
            aTimestamp === null &&
            bTimestamp === null
          ) {
            return 0;
          }

          if (aTimestamp === null) {
            return 1;
          }

          if (bTimestamp === null) {
            return -1;
          }

          return sortDir === "asc"
            ? aTimestamp -
                bTimestamp
            : bTimestamp -
                aTimestamp;
        },
      );
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
  const goDetail = useCallback(
    (id: string) => {
      navigate(
        `/mintRequest/${encodeURIComponent(
          id,
        )}`,
      );
    },
    [navigate],
  );

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
      onChange={
        setProductionFilter
      }
    />,
    <SortableTableHeader
      key="mintQuantity"
      label="Mint数量"
      sortKey="mintQuantity"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, direction) => {
        setSortKey(
          key as SortKey,
        );
        setSortDir(direction);
      }}
    />,
    <SortableTableHeader
      key="productionQuantity"
      label="生産量"
      sortKey="productionQuantity"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, direction) => {
        setSortKey(
          key as SortKey,
        );
        setSortDir(direction);
      }}
    />,
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(next: string[]) => {
        const mapped = (
          next ?? []
        )
          .map(
            asInspectionStatus,
          )
          .filter(
            (
              value,
            ): value is InspectionStatus =>
              value !== null,
          );

        setStatusFilter(mapped);
      }}
    />,
    <FilterableTableHeader
      key="requester"
      label="リクエスト者"
      options={requesterOptions}
      selected={requesterFilter}
      onChange={
        setRequesterFilter
      }
    />,
    <SortableTableHeader
      key="mintedAt"
      label="Mint実行日時"
      sortKey="mintedAt"
      activeKey={sortKey}
      direction={sortDir}
      onChange={(key, direction) => {
        setSortKey(
          key as SortKey,
        );
        setSortDir(direction);
      }}
    />,
  ];

  const onReset = useCallback(async () => {
    setTokenFilter([]);
    setProductionFilter([]);
    setRequesterFilter([]);
    setStatusFilter([]);
    setSortKey("mintedAt");
    setSortDir("desc");

    await fetchRows();
  }, [fetchRows]);

  const handleRowClick = useCallback(
    (id: string) => {
      goDetail(id);
    },
    [goDetail],
  );

  const handleRowKeyDown = useCallback(
    (
      event: React.KeyboardEvent<HTMLTableRowElement>,
      id: string,
    ) => {
      if (
        event.key === "Enter" ||
        event.key === " "
      ) {
        event.preventDefault();
        goDetail(id);
      }
    },
    [goDetail],
  );

  return {
    headers,
    rows,
    onReset,
    handleRowClick,
    handleRowKeyDown,
    loading,
    isResetting,
    error,
  };
};