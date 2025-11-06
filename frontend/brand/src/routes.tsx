import type { RouteObject } from "react-router-dom";
import BrandManagement from "./pages/brandManagement";

const routes: RouteObject[] = [
  { path: "", element: <BrandManagement /> },
  // 他のルート定義
];

export default routes;