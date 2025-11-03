import { Routes, Route, Navigate } from "react-router-dom";
import InquiryListPage from "./pages/InquiryManagementPage";

/**
 * InquiriesRoutes
 * 問い合わせモジュールのルート構成。
 * shell から import("inquiries/routes") でロードされる。
 */
export default function InquiriesRoutes() {
  return (
    <Routes>
      <Route path="/" element={<InquiryListPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
