// frontend/member/src/presentation/routes/routes.tsx

import React from "react";
import type { RouteObject } from "react-router-dom";
import MemberManagement from "../pages/memberManagement";
import MemberDetail from "../pages/memberDetail";

const routes: RouteObject[] = [
  {
    path: "",
    element: <MemberManagement />,
  },
  {
    // Member.id を URL パラメータとして利用
    path: ":memberId",
    element: <MemberDetail />,
  },
];

export default routes;
