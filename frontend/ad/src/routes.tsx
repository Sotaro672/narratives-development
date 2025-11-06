import type { RouteObject } from "react-router-dom";
import AdManagement from "./pages/adManagement";

const routes: RouteObject[] = [
  { path: "", element: <AdManagement /> },
  // 他のルート定義
];

export default routes;