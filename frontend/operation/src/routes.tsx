// frontend/operation/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenOperation from "./pages/tokenOperation";

/**
 * OperationsRoutes
 * 運用モジュールのルーティング定義。
 */
const routes: RouteObject[] = [
  { path: "", element: <TokenOperation /> },
  // 他のルート定義
];

export default routes;
