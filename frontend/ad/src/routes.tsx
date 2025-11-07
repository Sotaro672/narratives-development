import type { RouteObject } from "react-router-dom";
import AdManagement from "./pages/adManagement";
import AdDetail from "./pages/adDetail";
import AdCreate from "./pages/adCreate";

const routes: RouteObject[] = [
  { path: "", element: <AdManagement /> },
  { path: "create", element: <AdCreate /> },
  { path: ":campaign", element: <AdDetail /> },
  // 他のルート定義
];

export default routes;