// frontend/member/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import MemberManagement from "../pages/memberManagement";
import MemberDetail from "../pages/memberDetail";
import MemberCreate from "../pages/memberCreate";

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
  {
    path: "create",
    element: <MemberCreate />,
  },
];

export default routes;
