// frontend/productBlueprint/src/routes.tsx

import type { RouteObject } from "react-router-dom";

import ProductBlueprintManagement from "../../presentation/pages/productBlueprintManagement";
import ProductBlueprintDetail from "../../presentation/pages/productBlueprintDetail";
import ProductBlueprintCreate from "../../presentation/pages/productBlueprintCreate";

/**
 * Product Blueprint Module Routes
 * ベースパス: /productBlueprint
 *
 * - /productBlueprint            -> 一覧
 * - /productBlueprint/detail/:blueprintId -> 詳細
 * - /productBlueprint/create    -> 作成
 */
const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },
  { path: "detail/:blueprintId", element: <ProductBlueprintDetail /> },
  { path: "create", element: <ProductBlueprintCreate /> },
];

export default routes;
