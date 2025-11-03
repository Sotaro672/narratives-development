import { Routes, Route, Navigate } from "react-router-dom";
import OperationDashboardPage from "./pages/OperationDashboardPage";
import OperationTaskListPage from "./pages/OperationTaskListPage";
import OperationDetailPage from "./pages/OperationDetailPage";

/**
 * OperationsRoutes
 * 運用モジュールのルーティング定義。
 * shell から import("operations/routes") でロードされる。
 */
export default function OperationsRoutes() {
  return (
    <Routes>
      <Route path="/" element={<OperationDashboardPage />} />
      <Route path="/tasks" element={<OperationTaskListPage />} />
      <Route path="/:id" element={<OperationDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
