// frontend/console/sales/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import AnnouncementManagementPage from "../pages/announcementManagement";
import AnnouncementCreatePage from "../pages/announcementCreatePage";
import AnnouncementTokenListPage from "../pages/announcementTokenListPage";

/**
 * Sales Module Routes
 * - /sales
 * - /sales/create
 * - /sales/:tokenBlueprintId/create
 */
const routes: RouteObject[] = [
  { path: "", element: <AnnouncementManagementPage /> },
  { path: "create", element: <AnnouncementTokenListPage /> },
  { path: ":tokenBlueprintId/create", element: <AnnouncementCreatePage /> },
];

export default routes;