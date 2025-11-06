// frontend/production/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductionManagement from "./pages/productionManagement";
import ProductionDetail from "./pages/productionDetail";

const routes: RouteObject[] = [
  { path: "", element: <ProductionManagement /> },
  // 追加: /production/:productionId にマッチ
  { path: ":productionId", element: <ProductionDetail /> },

  // （必要なら残す）/production/plans で固定ページを出す場合はこれもOK
  // { path: "plans", element: <ProductionDetail /> },
];

export default routes;
