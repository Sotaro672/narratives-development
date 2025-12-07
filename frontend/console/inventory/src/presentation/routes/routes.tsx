// frontend\console\inventory\src\presentation\routes\routes.tsx
import type { RouteObject } from "react-router-dom";
import InventoryManagementPage from "../../presentation/pages/inventoryManagement";
import InventoryDetailPage from "../../presentation/pages/inventoryDetail";

/**
 * Inventory Module Routes
 * 在庫一覧・詳細ページのルーティング設定
 */
const routes: RouteObject[] = [
  // /inventory → 一覧
  { path: "", element: <InventoryManagementPage /> },

  // /inventory/detail/:inventoryId → 詳細
  { path: "detail/:inventoryId", element: <InventoryDetailPage /> },
];

export default routes;