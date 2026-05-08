import type { RouteObject } from "react-router-dom";
import ListManagement from "../../presentation/pages/listManagement";
import ListDetail from "../../presentation/pages/listDetail";

const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },

  { path: ":listId", element: <ListDetail /> },
];

export default routes;
