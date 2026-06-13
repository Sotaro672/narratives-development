import type { RouteObject } from "react-router-dom";
import AccountManagement from "../pages/accountManagement";

const routes: RouteObject[] = [
  { path: "", element: <AccountManagement /> },
  // 他のルート定義
];

export default routes;