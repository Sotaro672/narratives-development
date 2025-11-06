import type { RouteObject } from "react-router-dom";
import PermissionList from "./pages/PermissionList";

const routes: RouteObject[] = [
  { path: "", element: <PermissionList /> },
  // 他のルート定義
];

export default routes;
