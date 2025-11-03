import { Routes, Route, Navigate } from "react-router-dom";
import ProductionDashboardPage from "./pages/ProductionDashboardPage";
import ProductionPlanPage from "./pages/ProductionPlanPage";
import ProductionProgressPage from "./pages/ProductionProgressPage";
import ProductionDetailPage from "./pages/ProductionDetailPage";

/**
 * ProductionRoutes
 * 生産計画モジュールのルーティング構成。
 * shell から import("production/routes") でロードされる。
 */
export default function ProductionRoutes() {
  return (
    <Routes>
      <Route path="/" element={<ProductionDashboardPage />} />
      <Route path="/plans" element={<ProductionPlanPage />} />
      <Route path="/progress" element={<ProductionProgressPage />} />
      <Route path="/:id" element={<ProductionDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
