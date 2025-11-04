import { Routes, Route, Navigate } from "react-router-dom";
import InquiryManagementPage from "./pages/InquiryManagementPage";

/**
 * InquiriesRoutes
 * 問い合わせモジュールのルート構成。
 */
export default function InquiryRoutes() {
  return (
    <Routes>
      <Route path="/" element={<InquiryManagementPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
