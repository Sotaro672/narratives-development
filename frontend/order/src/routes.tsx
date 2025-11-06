// frontend/order/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import OrderManagement from "./pages/orderManagement";

const routes: RouteObject[] = [
  { path: "", element: <OrderManagement /> },
  // 他のルート定義
];

export default routes;