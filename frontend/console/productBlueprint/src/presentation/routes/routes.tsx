// frontend/productBlueprint/src/routes.tsx

import type { RouteObject } from "react-router-dom";

import ProductBlueprintManagement from "../../presentation/pages/productBlueprintManagement";
import ProductBlueprintDetail from "../../presentation/pages/productBlueprintDetail";
import ProductBlueprintCreate from "../../presentation/pages/productBlueprintCreate";
import ProductBlueprintDeleted from "../../presentation/pages/productBlueprintDeleted";

// ★ 新規追加
import ProductBlueprintDeletedDetail from "../../presentation/pages/productBlueprintDeletedDetail";

const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintManagement /> },

  // 削除済み一覧
  { path: "deleted", element: <ProductBlueprintDeleted /> },

  // ★ 削除済み詳細ページを追加
  { path: "deleted/detail/:blueprintId", element: <ProductBlueprintDeletedDetail /> },

  // 通常の詳細ページ
  { path: "detail/:blueprintId", element: <ProductBlueprintDetail /> },

  { path: "create", element: <ProductBlueprintCreate /> },
];

export default routes;
