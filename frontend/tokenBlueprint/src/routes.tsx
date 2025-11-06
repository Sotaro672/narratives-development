// frontend/tokenBlueprint/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenBlueprintManagement from "./pages/TokenBlueprintManagement";
//import TokenBlueprintDetail from "./pages/TokenBlueprintDetail";

/**
 * Token Blueprint Module Routes
 * - /token-blueprint
 * - /token-blueprint/:blueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <TokenBlueprintManagement /> },
  //{ path: ":blueprintId", element: <TokenBlueprintDetail /> },
];

export default routes;
