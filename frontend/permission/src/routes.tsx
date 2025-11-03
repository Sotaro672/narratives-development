import type { RouteObject } from "react-router-dom";
import PermissionManagementPage from "./pages/PermissionManagementPage";
import RoleDetailPage from "./pages/RoleDetailPage";

/**
 * Permission module routes
 * - /permission
 * - /permission/:roleId
 */
const routes: RouteObject[] = [
  { path: "", element: <PermissionManagementPage /> },
  { path: ":roleId", element: <RoleDetailPage /> },
];

export default routes;
