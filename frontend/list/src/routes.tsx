// frontend/list/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ListManagement from "./pages/listManagement";

/**
 * ListingsRoutes
 * 出品モジュールのルート構成。
 * shell から import("listings/routes") でロードされる。
 */
const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },
  // 他のルート定義
];

export default routes;
