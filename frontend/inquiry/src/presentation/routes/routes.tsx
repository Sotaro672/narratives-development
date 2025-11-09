// frontend/inquiry/src/routes.tsx
import type { RouteObject } from "react-router-dom";
import InquiryManagement from "../../presentation/pages/inquiryManagement";
import InquiryDetail from "../../presentation/pages/inquiryDetail";

const routes: RouteObject[] = [
  { path: "", element: <InquiryManagement /> },
  { path: "/inquiry/:inquiryId", element: <InquiryDetail /> },
];

export default routes;
