import type { RouteObject } from "react-router-dom";
import ProductBlueprintManagementPage from "./pages/ProductBlueprintManagementPage";
import ProductBlueprintDetailPage from "./pages/ProductBlueprintDetailPage";

/**
 * Product Blueprint Module Routes
 * - /product-blueprint
 * - /product-blueprint/:blueprintId
 */
const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagementPage /> },
  { path: ":blueprintId", element: <ProductBlueprintDetailPage /> },
];

export default routes;
