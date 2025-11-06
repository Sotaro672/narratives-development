// frontend/production/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductionManagement from "./pages/productionManagement";
//import ProductionDetail from "./pages/productionDetail";

/**
 * Production Module Routes
 * - /production
 * - /production/plans
 * Shell 側で:
 *   { path: "/production", children: productionRoutes }
 * のように取り込みます。
 */
const routes: RouteObject[] = [
  { path: "", element: <ProductionManagement /> },
  //{ path: "plans", element: <ProductionDetail /> },
];

export default routes;
