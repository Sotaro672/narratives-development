// frontend/admin/shell/src/features/report/hooks/useReportDetail.ts

import { useCallback, useEffect, useRef, useState } from "react";

import type {
  ReportActorType,
  ReportCase,
  ReportDecision,
  ReportItem,
  ReportItemSort,
  ReportReason,
  ReportSortOrder,
} from "../../../../shared/type/report";
import { useReportPending } from "../../context/ReportPendingContext";
import {
  decideReport,
  getReport,
} from "../../infrastructure/reportApi";

const DEFAULT_PAGE = 1;
const DEFAULT_PER_PAGE = 50;
const DEFAULT_SORT: ReportItemSort = "createdAt";
const DEFAULT_ORDER: ReportSortOrder = "desc";

export function useReportDetail(caseId: string | undefined) {
  const { refreshPendingCount } = useReportPending();

  const requestIdRef = useRef(0);
  const decisionRequestIdRef = useRef(0);

  const [reportCase, setReportCase] = useState<ReportCase | null>(null);
  const [reports, setReports] = useState<ReportItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [reporterType, setReporterTypeState] = useState<ReportActorType | undefined>(undefined);
  const [reporterId, setReporterIdState] = useState("");
  const [companyId, setCompanyIdState] = useState("");
  const [reason, setReasonState] = useState<ReportReason | undefined>(undefined);
  const [page, setPageState] = useState(DEFAULT_PAGE);
  const [perPage, setPerPageState] = useState(DEFAULT_PER_PAGE);
  const [sort, setSortState] = useState<ReportItemSort>(DEFAULT_SORT);
  const [order, setOrderState] = useState<ReportSortOrder>(DEFAULT_ORDER);
  const [totalCount, setTotalCount] = useState(0);
  const [totalPages, setTotalPages] = useState(0);

  const [deciding, setDeciding] = useState(false);
  const [decisionError, setDecisionError] = useState<string | null>(null);

  const loadReport = useCallback(async () => {
    const normalizedCaseId = caseId?.trim() ?? "";
    const requestId = ++requestIdRef.current;

    if (!normalizedCaseId) {
      setReportCase(null);
      setReports([]);
      setTotalCount(0);
      setTotalPages(0);
      setLoading(false);
      setError("通報ケースIDが指定されていません。");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await getReport(normalizedCaseId, {
        reporterType,
        reporterId: reporterId.trim() || undefined,
        companyId: companyId.trim() || undefined,
        reason,
        page,
        perPage,
        sort,
        order,
      });

      if (requestId !== requestIdRef.current) {
        return;
      }

      setReportCase(result.case);
      setReports(result.reports.items);
      setTotalCount(result.reports.totalCount);
      setTotalPages(result.reports.totalPages);

      if (result.reports.page !== page) {
        setPageState(result.reports.page);
      }

      if (result.reports.perPage !== perPage) {
        setPerPageState(result.reports.perPage);
      }
    } catch (cause) {
      if (requestId !== requestIdRef.current) {
        return;
      }

      setReportCase(null);
      setReports([]);
      setTotalCount(0);
      setTotalPages(0);
      setError(
        cause instanceof Error
          ? cause.message
          : "通報詳細の取得に失敗しました。",
      );
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [
    caseId,
    reporterType,
    reporterId,
    companyId,
    reason,
    page,
    perPage,
    sort,
    order,
  ]);

  useEffect(() => {
    void loadReport();

    return () => {
      requestIdRef.current += 1;
    };
  }, [loadReport]);

  useEffect(() => {
    decisionRequestIdRef.current += 1;
    setDeciding(false);
    setDecisionError(null);
  }, [caseId]);

  const setReporterType = useCallback(
    (value: ReportActorType | undefined) => {
      setReporterTypeState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
  );

  const setReporterId = useCallback((value: string) => {
    setReporterIdState(value);
    setPageState(DEFAULT_PAGE);
  }, []);

  const setCompanyId = useCallback((value: string) => {
    setCompanyIdState(value);
    setPageState(DEFAULT_PAGE);
  }, []);

  const setReason = useCallback(
    (value: ReportReason | undefined) => {
      setReasonState(value);
      setPageState(DEFAULT_PAGE);
    },
    [],
  );

  const setPage = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }

    setPageState(Math.max(1, Math.trunc(value)));
  }, []);

  const setPerPage = useCallback((value: number) => {
    if (!Number.isFinite(value)) {
      return;
    }

    setPerPageState(Math.max(1, Math.min(200, Math.trunc(value))));
    setPageState(DEFAULT_PAGE);
  }, []);

  const setSort = useCallback((value: ReportItemSort) => {
    setSortState(value);
    setPageState(DEFAULT_PAGE);
  }, []);

  const setOrder = useCallback((value: ReportSortOrder) => {
    setOrderState(value);
    setPageState(DEFAULT_PAGE);
  }, []);

  const resetFilters = useCallback(() => {
    setReporterTypeState(undefined);
    setReporterIdState("");
    setCompanyIdState("");
    setReasonState(undefined);
    setPageState(DEFAULT_PAGE);
    setPerPageState(DEFAULT_PER_PAGE);
    setSortState(DEFAULT_SORT);
    setOrderState(DEFAULT_ORDER);
  }, []);

  const reload = useCallback(async () => {
    await loadReport();
  }, [loadReport]);

  const decide = useCallback(
    async (
      decision: ReportDecision,
      decisionReason: string,
    ): Promise<ReportCase | null> => {
      const normalizedCaseId = caseId?.trim() ?? "";
      const normalizedReason = decisionReason.trim();

      if (!normalizedCaseId) {
        setDecisionError("通報ケースIDが指定されていません。");
        return null;
      }

      if (!normalizedReason) {
        setDecisionError("裁定理由を入力してください。");
        return null;
      }

      if (deciding) {
        return null;
      }

      const requestId = ++decisionRequestIdRef.current;

      setDeciding(true);
      setDecisionError(null);

      try {
        const updatedCase = await decideReport(normalizedCaseId, {
          decision,
          reason: normalizedReason,
        });

        if (requestId !== decisionRequestIdRef.current) {
          return null;
        }

        setReportCase(updatedCase);
        void refreshPendingCount();

        return updatedCase;
      } catch (cause) {
        if (requestId !== decisionRequestIdRef.current) {
          return null;
        }

        setDecisionError(
          cause instanceof Error
            ? cause.message
            : "通報の裁定に失敗しました。",
        );
        return null;
      } finally {
        if (requestId === decisionRequestIdRef.current) {
          setDeciding(false);
        }
      }
    },
    [caseId, deciding, refreshPendingCount],
  );

  const keep = useCallback(
    async (decisionReason: string) => {
      return decide("KEEP", decisionReason);
    },
    [decide],
  );

  const remove = useCallback(
    async (decisionReason: string) => {
      return decide("REMOVE", decisionReason);
    },
    [decide],
  );

  const hasPreviousPage = page > 1;
  const hasNextPage = totalPages > 0 && page < totalPages;
  const canKeep = reportCase?.status === "PENDING" && !deciding;
  const canRemove =
    (reportCase?.status === "PENDING" || reportCase?.status === "KEPT") &&
    !deciding;
  const canDecide = canKeep || canRemove;

  return {
    reportCase,
    reports,
    loading,
    error,
    reporterType,
    reporterId,
    companyId,
    reason,
    page,
    perPage,
    sort,
    order,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    deciding,
    decisionError,
    canDecide,
    canKeep,
    canRemove,
    setReporterType,
    setReporterId,
    setCompanyId,
    setReason,
    setPage,
    setPerPage,
    setSort,
    setOrder,
    resetFilters,
    reload,
    decide,
    keep,
    remove,
  };
}