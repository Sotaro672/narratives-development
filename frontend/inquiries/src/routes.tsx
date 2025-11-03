import { Routes, Route, Navigate } from "react-router-dom";
import InquiryListPage from "./pages/InquiryListPage";
import InquiryDetailPage from "./pages/InquiryDetailPage";
import InquiryAssignPage from "./pages/InquiryAssignPage";

/**
 * InquiriesRoutes
 * 問い合わせモジュールのルート構成。
 * shell から import("inquiries/routes") でロードされる。
 */
export default function InquiriesRoutes() {
  return (
    <Routes>
      <Route path="/" element={<InquiryListPage />} />
      <Route path="/:id" element={<InquiryDetailPage />} />
      <Route path="/assign/:id" element={<InquiryAssignPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
