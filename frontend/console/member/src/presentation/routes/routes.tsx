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
    /**
     * IMPORTANT:
     * この URL パラメータは Firestore member docId ではなく Firebase Auth UID。
     *
     * backend:
     * - GET /members/{uid} は Firebase UID 専用
     * - PATCH /members/{docId} は Firestore member docId 専用
     */
    path: ":memberUid",
    element: <MemberDetail />,
  },
  {
    path: "create",
    element: <MemberCreate />,
  },
];

export default routes;