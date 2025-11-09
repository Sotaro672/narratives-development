// frontend/operation/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenOperation from "../../presentation/pages/tokenOperation";
import TokenOperationDetail from "../../presentation/pages/tokenOperationDetail";

/**
 * OperationsRoutes
 * 運用モジュールのルーティング定義。
 */
const routes: RouteObject[] = [
  { path: "", element: <TokenOperation /> },
  { path: ":tokenOperationId", element: <TokenOperationDetail /> },
  // 他のルート定義
];

export default routes;
