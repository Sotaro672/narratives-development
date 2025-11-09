// frontend/member/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import React from "react";

import MemberManagement from "../pages/memberManagement";
import MemberDetail from "../pages/memberDetail";

const routes: RouteObject[] = [
  { path: "", element: <MemberManagement /> },
  { path: ":email", element: <MemberDetail /> },
  // 他のルート定義
];

export default routes;
