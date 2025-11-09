import type { RouteObject } from "react-router-dom";
import PermissionList from "../../presentation/pages/permissionList";
import PermissionDetail from "../../presentation/pages/permissionDetail";

const routes: RouteObject[] = [
  { path: "", element: <PermissionList /> },
  { path: ":permissionId", element: <PermissionDetail /> },
  // 他のルート定義
];

export default routes;
