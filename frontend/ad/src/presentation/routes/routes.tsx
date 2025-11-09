import type { RouteObject } from "react-router-dom";
import AdManagement from "../../presentation/pages/adManagement";
import AdDetail from "../../presentation/pages/adDetail";
import AdCreate from "../../presentation/pages/adCreate";

const routes: RouteObject[] = [
  { path: "", element: <AdManagement /> },
  { path: "create", element: <AdCreate /> },
  { path: ":campaign", element: <AdDetail /> },
  // 他のルート定義
];

export default routes;