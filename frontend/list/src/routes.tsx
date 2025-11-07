// frontend/list/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ListManagement from "./pages/listManagement";
import ListDetail from "./pages/listDetail";
import ListCreate from "./pages/listCreate";

/**
 * ListingsRoutes
 * 出品モジュールのルート構成。
 * shell から import("listings/routes") でロードされる。
 */
const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },
  { path: ":listId", element: <ListDetail /> },
  { path: "create", element: <ListCreate /> },
  // 他のルート定義
];

export default routes;
