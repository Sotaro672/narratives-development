// frontend/announce/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import AnnounceManagement from "../pages/announceManagement";

const routes: RouteObject[] = [
  { path: "/announcement", element: <AnnounceManagement /> },
];

export default routes;

