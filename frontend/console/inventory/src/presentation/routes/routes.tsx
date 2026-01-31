// frontend/console/inventory/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import InventoryManagementPage from "../../presentation/pages/inventoryManagement";
import InventoryDetailPage from "../../presentation/pages/inventoryDetail";

// ✅ NEW: inventory ドメイン内の出品作成（仮）ページ
import InventoryListCreatePage from "../pages/listCreate";

/**
 * Inventory Module Routes
 *
 * 方針A:
 * - URL に productBlueprintId + tokenBlueprintId を渡す
 * - 詳細側で inventoryIds を解決して表示する
 *
 */
const routes: RouteObject[] = [
  // /inventory → 一覧
  { path: "", element: <InventoryManagementPage /> },

  // ✅ /inventory/detail/:productBlueprintId/:tokenBlueprintId → 詳細（方針A）
  {
    path: "detail/:productBlueprintId/:tokenBlueprintId",
    element: <InventoryDetailPage />,
  },

  // ✅ /inventory/list/create/:inventoryId → 出品作成（inventoryId を直接渡す）
  {
    path: "list/create/:inventoryId",
    element: <InventoryListCreatePage />,
  },

  // ✅ /inventory/list/create/:productBlueprintId/:tokenBlueprintId → 出品作成（pb/tb を渡す）
  {
    path: "list/create/:productBlueprintId/:tokenBlueprintId",
    element: <InventoryListCreatePage />,
  },
];

export default routes;
