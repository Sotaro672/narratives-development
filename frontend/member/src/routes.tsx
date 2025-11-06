// frontend/member/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import React from "react";

import MemberManagement from "./pages/memberManagement";

const routes: RouteObject[] = [
  { path: "", element: <MemberManagement /> },
  // 他のルート定義
];

export default routes;
