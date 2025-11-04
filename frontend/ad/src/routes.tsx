import { Routes, Route, Navigate } from "react-router-dom";
import AdsDashboardPage from "./pages/AdsDashboardPage";
import AdsCampaignPage from "./pages/AdsCampaignPage";
import AdsReportPage from "./pages/AdsReportPage";
import AdsDetailPage from "./pages/AdsDetailPage";

/**
 * AdsRoutes
 * 広告運用モジュールのルーティング構成。
 * shell から import("ads/routes") でロードされる。
 */
export default function AdsRoutes() {
  return (
    <Routes>
      <Route path="/" element={<AdsDashboardPage />} />
      <Route path="/campaigns" element={<AdsCampaignPage />} />
      <Route path="/reports" element={<AdsReportPage />} />
      <Route path="/:id" element={<AdsDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
