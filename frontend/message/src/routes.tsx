//frontend\message\src\routes.tsx
import type { RouteObject } from "react-router-dom";
import MessageManagement from "./pages/messageManagement";

const routes: RouteObject[] = [
  { path: "/message", element: <MessageManagement /> },
];

export default routes;