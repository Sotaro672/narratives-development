import type { RouteObject } from "react-router-dom";
import CompanyManagementPage from "./pages/CompanyManagementPage";
import CompanyDetailPage from "./pages/CompanyDetailPage";

/**
 * Company Module Routes
 * - /company
 * - /company/:companyId
 */
const routes: RouteObject[] = [
  { path: "", element: <CompanyManagementPage /> },
  { path: ":companyId", element: <CompanyDetailPage /> },
];

export default routes;
