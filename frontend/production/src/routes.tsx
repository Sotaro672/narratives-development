// frontend/production/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductionManagement from "./pages/productionManagement";
import ProductionDetail from "./pages/productionDetail";
import ProductionCreate from "./pages/productionCreate";

const routes: RouteObject[] = [
  { path: "", element: <ProductionManagement /> },
  { path: ":productionId", element: <ProductionDetail /> },
  { path: "create", element: <ProductionCreate /> },
];

export default routes;
