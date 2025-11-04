import { Routes, Route, Navigate } from "react-router-dom";
import MintDashboardPage from "./pages/MintDashboardPage";
import MintTokenPage from "./pages/MintTokenPage";
import MintHistoryPage from "./pages/MintHistoryPage";
import MintDetailPage from "./pages/MintDetailPage";

/**
 * MintRoutes
 * トークンミントモジュールのルーティング構成。
 * shell から import("mint/routes") でロードされる。
 */
export default function MintRoutes() {
  return (
    <Routes>
      <Route path="/" element={<MintDashboardPage />} />
      <Route path="/token" element={<MintTokenPage />} />
      <Route path="/history" element={<MintHistoryPage />} />
      <Route path="/:id" element={<MintDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
