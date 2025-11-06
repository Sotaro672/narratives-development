import type { RouteObject } from "react-router-dom";
import InquiryManagement from "./pages/InquiryManagement";

const routes: RouteObject[] = [
  { path: "", element: <InquiryManagement /> },
];

export default routes;
