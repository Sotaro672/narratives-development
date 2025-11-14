// frontend\tokenBlueprint\src\presentation\routes\routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenBlueprintManagement from "../pages/TokenBlueprintManagement";
import TokenBlueprintDetail from "../pages/tokenBlueprintDetail";
import TokenBlueprintCreate from "../pages/tokenBlueprintCreate";

/**
 * Token Blueprint Module Routes
 * - /token-blueprint
 * - /token-blueprint/:blueprintId
 * - /token-blueprint/create
 */
const routes: RouteObject[] = [
  { path: "", element: <TokenBlueprintManagement /> },
  { path: ":tokenBlueprintId", element: <TokenBlueprintDetail /> },
  { path: "create", element: <TokenBlueprintCreate /> },
];

export default routes;
