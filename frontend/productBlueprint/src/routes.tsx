// frontend/productBlueprint/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductBlueprintManagement from "./pages/productBlueprintManagement";
import ProductBlueprintDetail from "./pages/productBlueprintDetail";
import ProductBlueprintCreate from "./pages/productBlueprintCreate";

/**
 * Product Blueprint Module Routes
 * - /product-blueprint
 * - /product-blueprint/:blueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },
  { path: ":blueprintId", element: <ProductBlueprintDetail /> },
  { path: "create", element: <ProductBlueprintCreate /> },
];

export default routes;
