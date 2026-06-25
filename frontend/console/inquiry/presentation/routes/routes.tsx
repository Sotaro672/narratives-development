// frontend/inquiry/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import InquiryManagement from "../pages/inquiryManagement";
import InquiryDetail from "../pages/inquiryDetail";

const routes: RouteObject[] = [
  { path: "", element: <InquiryManagement /> },
  { path: ":inquiryId", element: <InquiryDetail /> },
];

export default routes;