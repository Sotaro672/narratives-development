import type { RouteObject } from "react-router-dom";
import React from "react";

import MemberManagementPage from "./pages/MemberManagementPage";
import MemberDetailPage from "./pages/MemberDetailPage";

/**
 * Member Module Routes
 * ------------------------------------------------------------------
 * 組織メンバー管理のルーティング構成
 * 
 * - /member              → メンバー一覧ページ
 * - /member/:memberId    → メンバー詳細ページ
 * 
 * shell 側で Module Federation 経由で lazy import される。
 * 例: route-manifest.ts → member: "member/routes"
 * ------------------------------------------------------------------
 */
const routes: RouteObject[] = [
  {
    path: "",
    element: <MemberManagementPage />,
  },
  {
    path: ":memberId",
    element: <MemberDetailPage />,
  },
];

export default routes;
