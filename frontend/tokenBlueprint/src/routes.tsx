import type { RouteObject } from "react-router-dom";
import TokenBlueprintManagementPage from "./pages/TokenBlueprintManagementPage";
import TokenBlueprintDetailPage from "./pages/TokenBlueprintDetailPage";

/**
 * Token Blueprint Module Routes
 * - /token-blueprint
 * - /token-blueprint/:blueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <TokenBlueprintManagementPage /> },
  { path: ":blueprintId", element: <TokenBlueprintDetailPage /> },
];

export default routes;
