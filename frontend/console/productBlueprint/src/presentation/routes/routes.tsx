// frontend/productBlueprint/src/routes.tsx

import type { RouteObject } from "react-router-dom";

import ProductBlueprintManagement from "../../presentation/pages/productBlueprintManagement";
import ProductBlueprintDetail from "../../presentation/pages/productBlueprintDetail";
import ProductBlueprintCreate from "../../presentation/pages/productBlueprintCreate";
import ProductBlueprintDeleted from "../../presentation/pages/productBlueprintDeleted";
const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },
  { path: "deleted", element: <ProductBlueprintDeleted /> }, // ★追加
  { path: "detail/:blueprintId", element: <ProductBlueprintDetail /> },
  { path: "create", element: <ProductBlueprintCreate /> },
];

export default routes;
