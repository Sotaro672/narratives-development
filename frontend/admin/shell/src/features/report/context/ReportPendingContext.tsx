// frontend/admin/shell/src/features/report/context/ReportPendingContext.tsx

import {
  createContext,
  type ReactNode,
  useContext,
  useMemo,
} from "react";

import { useReportPendingCount } from "../hooks/useReportPendingCount";

type ReportPendingContextValue = {
  pendingCount: number;
  loading: boolean;
  error: string | null;
  refreshPendingCount: () => Promise<void>;
};

type ReportPendingProviderProps = {
  children: ReactNode;
};

const ReportPendingContext =
  createContext<ReportPendingContextValue | null>(null);

export function ReportPendingProvider({
  children,
}: ReportPendingProviderProps) {
  const {
    pendingCount,
    loading,
    error,
    refreshPendingCount,
  } = useReportPendingCount();

  const value = useMemo<ReportPendingContextValue>(
    () => ({
      pendingCount,
      loading,
      error,
      refreshPendingCount,
    }),
    [
      pendingCount,
      loading,
      error,
      refreshPendingCount,
    ],
  );

  return (
    <ReportPendingContext.Provider value={value}>
      {children}
    </ReportPendingContext.Provider>
  );
}

export function useReportPending(): ReportPendingContextValue {
  const context = useContext(ReportPendingContext);

  if (!context) {
    throw new Error(
      "useReportPending must be used within ReportPendingProvider.",
    );
  }

  return context;
}