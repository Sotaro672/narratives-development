import type { RouteObject } from "react-router-dom";
import ListManagement from "../../presentation/pages/listManagement";
import ListDetail from "../../presentation/pages/listDetail";
import ListCreate from "../../presentation/pages/listCreate";

const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },

  // ✅ create（素のcreate）
  { path: "create", element: <ListCreate /> },

  // ✅ create に inventoryId を渡すルート（推奨）
  { path: "create/:inventoryId", element: <ListCreate /> },

  // ✅ create に pbId/tbId を渡すルート（必要なら）
  { path: "create/:productBlueprintId/:tokenBlueprintId", element: <ListCreate /> },

  { path: ":listId", element: <ListDetail /> },
];

export default routes;
