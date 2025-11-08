// frontend/announce/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import AnnounceManagement from "./pages/announceManagement";


const routes: RouteObject[] = [
  { path: "", element: <AnnounceManagement /> },
  // 他のルート定義
];

export default routes;
