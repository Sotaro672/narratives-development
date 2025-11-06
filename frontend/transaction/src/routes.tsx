import type { RouteObject } from "react-router-dom";
import TransactionsList from "./pages/transactionList";

const routes: RouteObject[] = [
  { path: "", element: <TransactionsList /> },
  // 他のルート定義
];

export default routes;