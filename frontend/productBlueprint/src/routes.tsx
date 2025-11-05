import type { RouteObject } from "react-router-dom";
import ProductBlueprintManagement from "./pages/productBlueprintManagement";
import ProductBlueprintDetail from "./pages/productBlueprintDetail";

/**
 * Product Blueprint Module Routes
 * - /product-blueprint
 * - /product-blueprint/:blueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },
  { path: ":blueprintId", element: <ProductBlueprintDetail /> },
];

export default routes;
