import { Routes, Route, Navigate } from "react-router-dom";
import DesignDashboardPage from "./pages/DesignDashboardPage";
import DesignProjectListPage from "./pages/DesignProjectListPage";
import DesignDetailPage from "./pages/DesignDetailPage";
import DesignReviewPage from "./pages/DesignReviewPage";

/**
 * DesignRoutes
 * 設計モジュールのルーティング構成。
 * shell から import("design/routes") でロードされる。
 */
export default function DesignRoutes() {
  return (
    <Routes>
      <Route path="/" element={<DesignDashboardPage />} />
      <Route path="/projects" element={<DesignProjectListPage />} />
      <Route path="/:id" element={<DesignDetailPage />} />
      <Route path="/review/:id" element={<DesignReviewPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
