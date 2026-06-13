import type { RouteObject } from "react-router-dom";
import ListManagement from "../pages/listManagement";
import ListDetail from "../pages/listDetail";

const routes: RouteObject[] = [
  { path: "", element: <ListManagement /> },

  { path: ":listId", element: <ListDetail /> },
];

export default routes;
