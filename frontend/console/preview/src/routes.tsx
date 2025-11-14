import { Routes, Route, Navigate } from "react-router-dom";
import PreviewDashboardPage from "./pages/PreviewDashboardPage";
import PreviewDetailPage from "./pages/PreviewDetailPage";
import PreviewComparePage from "./pages/PreviewComparePage";

/**
 * PreviewRoutes
 * プレビューモジュールのルート構成。
 * shell から import("preview/routes") でロードされる。
 */
export default function PreviewRoutes() {
  return (
    <Routes>
      <Route path="/" element={<PreviewDashboardPage />} />
      <Route path="/:id" element={<PreviewDetailPage />} />
      <Route path="/compare/:id" element={<PreviewComparePage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
