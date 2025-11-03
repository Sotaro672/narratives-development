import type { RouteObject } from "react-router-dom";
import InventoryManagementPage from "./pages/InventoryManagementPage";
import InventoryDetailPage from "./pages/InventoryDetailPage";

/**
 * Inventory Module Routes
 * 在庫一覧・詳細ページのルーティング設定
 */
const routes: RouteObject[] = [
  { path: "", element: <InventoryManagementPage /> },
  { path: ":inventoryId", element: <InventoryDetailPage /> },
];

export default routes;
