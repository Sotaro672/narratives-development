// frontend/list/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ListManagement from "./pages/listManagement";
import ListDetail from "./pages/listDetail";

/**
 * ListingsRoutes
 * 出品モジュールのルート構成。
 * shell から import("listings/routes") でロードされる。
 */
const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },
  { path: ":listId", element: <ListDetail /> },
  // 他のルート定義
];

export default routes;
