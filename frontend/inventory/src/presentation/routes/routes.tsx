// frontend/inventory/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import InventoryManagement from "./pages/inventoryManagement";
import InventoryDetail from "./pages/inventoryDetail";

/**
 * Inventory Module Routes
 * 在庫一覧・詳細ページのルーティング設定
 */
const routes: RouteObject[] = [
  { path: "", element: <InventoryManagement /> },
  { path: ":inventoryId", element: <InventoryDetail /> },
];

export default routes;
