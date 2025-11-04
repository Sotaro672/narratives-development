import { Routes, Route, Navigate } from "react-router-dom";
import TransactionsDashboardPage from "./pages/TransactionsDashboardPage";
import TransactionDetailPage from "./pages/TransactionDetailPage";
import TransactionHistoryPage from "./pages/TransactionHistoryPage";
import TransactionReportPage from "./pages/TransactionReportPage";

/**
 * TransactionsRoutes
 * 取引・決済モジュールのルーティング構成。
 * shell から import("transactions/routes") でロードされる。
 */
export default function TransactionsRoutes() {
  return (
    <Routes>
      <Route path="/" element={<TransactionsDashboardPage />} />
      <Route path="/history" element={<TransactionHistoryPage />} />
      <Route path="/reports" element={<TransactionReportPage />} />
      <Route path="/:id" element={<TransactionDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
