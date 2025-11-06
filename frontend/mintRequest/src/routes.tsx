// frontend/mintRequest/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import MintRequestManagement from "./pages/mintRequestManagement";
import MintRequestDetail from "./pages/mintRequestDetail";

/**
 * MintRoutes
 * トークンミントモジュールのルーティング構成。
 * shell から import("mint/routes") でロードされる。
 */
const routes: RouteObject[] = [
  { path: "", element: <MintRequestManagement /> },
  { path: ":requestId", element: <MintRequestDetail /> },
];

export default routes;
