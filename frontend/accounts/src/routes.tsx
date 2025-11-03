import { Routes, Route, Navigate } from "react-router-dom";
import AccountsDashboardPage from "./pages/AccountsDashboardPage";
import AccountsTransactionListPage from "./pages/AccountsTransactionListPage";
import AccountsReportPage from "./pages/AccountsReportPage";
import AccountsDetailPage from "./pages/AccountsDetailPage";

/**
 * AccountsRoutes
 * 会計・財務モジュールのルーティング構成。
 * shell から import("accounts/routes") でロードされる。
 */
export default function AccountsRoutes() {
  return (
    <Routes>
      <Route path="/" element={<AccountsDashboardPage />} />
      <Route path="/transactions" element={<AccountsTransactionListPage />} />
      <Route path="/reports" element={<AccountsReportPage />} />
      <Route path="/:id" element={<AccountsDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
