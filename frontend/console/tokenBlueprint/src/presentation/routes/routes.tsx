// frontend/console/tokenBlueprint/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenBlueprintManagement from "../pages/tokenBlueprintManagement";
import TokenBlueprintDetail from "../pages/tokenBlueprintDetail";
import TokenBlueprintCreate from "../pages/tokenBlueprintCreate";

const routes: RouteObject[] = [
  { path: "", element: <TokenBlueprintManagement /> },
  { path: ":tokenBlueprintId", element: <TokenBlueprintDetail /> },
  { path: "create", element: <TokenBlueprintCreate /> },
];

export default routes;