import type { RouteObject } from "react-router-dom";
import BrandManagementPage from "./pages/BrandManagementPage";
import BrandDetailPage from "./pages/BrandDetailPage";

const routes: RouteObject[] = [
  { path: "", element: <BrandManagementPage /> },
  { path: ":brandId", element: <BrandDetailPage /> },
];

export default routes;
