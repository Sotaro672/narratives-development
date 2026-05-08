// frontend/productBlueprint/src/routes.tsx

import type { RouteObject } from "react-router-dom";

import ProductBlueprintManagement from "../../presentation/pages/productBlueprintManagement";
import ProductBlueprintDetail from "../../presentation/pages/productBlueprintDetail";
import ProductBlueprintCreate from "../../presentation/pages/productBlueprintCreate";

const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },

  // 通常の詳細ページ
  { path: "detail/:blueprintId", element: <ProductBlueprintDetail /> },

  { path: "create", element: <ProductBlueprintCreate /> },
];

export default routes;