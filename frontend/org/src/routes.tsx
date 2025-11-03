import { Routes, Route, Navigate } from "react-router-dom";
import OrganizationOverview from "./pages/OrganizationOverview";
import MemberManagementPage from "./pages/MemberManagementPage";
import BrandManagementPage from "./pages/BrandManagementPage";
import PermissionSettingsPage from "./pages/PermissionSettingsPage";

/**
 * OrgRoutes
 * ------------------------------------------------------------------------
 * このモジュールは、組織管理モジュール（org）のルーティング定義を提供する。
 * Shell 側では `import("org/routes")` により遅延ロードされる。
 * ------------------------------------------------------------------------
 */
export default function OrgRoutes() {
  return (
    <Routes>
      {/* ───────────────────────────────
          組織概要（デフォルト）
      ─────────────────────────────── */}
      <Route path="/" element={<OrganizationOverview />} />

      {/* ───────────────────────────────
          メンバー管理
      ─────────────────────────────── */}
      <Route path="/members" element={<MemberManagementPage />} />

      {/* ───────────────────────────────
          ブランド管理
      ─────────────────────────────── */}
      <Route path="/brands" element={<BrandManagementPage />} />

      {/* ───────────────────────────────
          権限設定
      ─────────────────────────────── */}
      <Route path="/permissions" element={<PermissionSettingsPage />} />

      {/* ───────────────────────────────
          その他（リダイレクト）
      ─────────────────────────────── */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
