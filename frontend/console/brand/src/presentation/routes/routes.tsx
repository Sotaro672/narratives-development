import type { RouteObject } from "react-router-dom";
import BrandManagement from "../pages/brandManagement";
import BrandCreate from "../pages/brandCreate";
import BrandDetail from "../pages/brandDetail";

const routes: RouteObject[] = [
  { path: "", element: <BrandManagement /> },
  { path: "create", element: <BrandCreate /> },
  { path: ":brandId", element: <BrandDetail /> },
  // 他のルート定義
];

export default routes;