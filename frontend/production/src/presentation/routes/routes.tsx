// frontend/production/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductionManagement from "../../presentation/pages/productionManagement";
import ProductionDetail from "../../presentation/pages/productionDetail";
import ProductionCreate from "../../presentation/pages/productionCreate";

const routes: RouteObject[] = [
  { path: "", element: <ProductionManagement /> },
  { path: ":productionId", element: <ProductionDetail /> },
  { path: "create", element: <ProductionCreate /> },
];

export default routes;
