// frontend\console\sales\src\presentation\routes\routes.tsx
import type { RouteObject } from "react-router-dom";
import SalesManagement from "../pages/salesManagement";
import SalesDetail from "../pages/salesDetail";

/**
 * Sales Module Routes
 * - /sales
 * - /sales/:tokenBlueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <SalesManagement /> },
  { path: ":tokenBlueprintId", element: <SalesDetail /> },
];

export default routes;