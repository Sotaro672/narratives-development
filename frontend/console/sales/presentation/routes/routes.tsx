// frontend\console\sales\src\presentation\routes\routes.tsx
import type { RouteObject } from "react-router-dom";
import SalesManagement from "../pages/announcementManagement";
import SalesDetail from "../pages/announcementCreatePage";
import SalesCreate from "../pages/announcementTokenListPage";

/**
 * Sales Module Routes
 * - /sales
 * - /sales/create
 * - /sales/:tokenBlueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <SalesManagement /> },
  { path: "create", element: <SalesCreate /> },
  { path: ":tokenBlueprintId", element: <SalesDetail /> },
];

export default routes;