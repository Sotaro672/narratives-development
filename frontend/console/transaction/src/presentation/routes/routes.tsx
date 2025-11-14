import type { RouteObject } from "react-router-dom";
import TransactionsList from "../../presentation/pages/transactionList";
import TransactionDetail from "../../presentation/pages/transactionDetail";

const routes: RouteObject[] = [
  { path: "", element: <TransactionsList /> },
  { path: "/transaction/:transactionId", element: <TransactionDetail /> },
  // 他のルート定義
];

export default routes;