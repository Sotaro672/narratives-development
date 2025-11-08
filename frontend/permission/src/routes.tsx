import type { RouteObject } from "react-router-dom";
import PermissionList from "./pages/permissionList";
import PermissionDetail from "./pages/permissionDetail";

const routes: RouteObject[] = [
  { path: "", element: <PermissionList /> },
  { path: ":permissionId", element: <PermissionDetail /> },
  // 他のルート定義
];

export default routes;
