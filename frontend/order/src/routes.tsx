import { Routes, Route, Navigate } from "react-router-dom";
import OrdersDashboardPage from "./pages/OrdersDashboardPage";
import OrderDetailPage from "./pages/OrderDetailPage";
import OrderTrackingPage from "./pages/OrderTrackingPage";
import OrderHistoryPage from "./pages/OrderHistoryPage";

/**
 * OrdersRoutes
 * 注文モジュールのルーティング構成。
 * shell から import("orders/routes") でロードされる。
 */
export default function OrdersRoutes() {
  return (
    <Routes>
      <Route path="/" element={<OrdersDashboardPage />} />
      <Route path="/history" element={<OrderHistoryPage />} />
      <Route path="/tracking/:id" element={<OrderTrackingPage />} />
      <Route path="/:id" element={<OrderDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
