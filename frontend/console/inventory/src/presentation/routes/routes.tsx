// frontend/console/inventory/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import InventoryManagementPage from "../../presentation/pages/inventoryManagement";
import InventoryDetailPage from "../../presentation/pages/inventoryDetail";

/**
 * Inventory Module Routes
 * 在庫一覧・詳細ページのルーティング設定
 *
 * 方針A:
 * - URL に productBlueprintId + tokenBlueprintId を渡す
 * - 詳細側で inventoryIds を解決して表示する
 */
const routes: RouteObject[] = [
  // /inventory → 一覧
  { path: "", element: <InventoryManagementPage /> },

  // ✅ /inventory/detail/:productBlueprintId/:tokenBlueprintId → 詳細（方針A）
  {
    path: "detail/:productBlueprintId/:tokenBlueprintId",
    element: <InventoryDetailPage />,
  },
];

export default routes;
