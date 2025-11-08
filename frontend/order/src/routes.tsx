// frontend/order/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import OrderManagement from "./pages/orderManagement";
import OrderDetail from "./pages/orderDetail";

const routes: RouteObject[] = [
  { path: "", element: <OrderManagement /> },
  { path: ":orderId", element: <OrderDetail /> },
  // 他のルート定義
];

export default routes;