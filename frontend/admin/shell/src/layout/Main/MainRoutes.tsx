// frontend/admin/shell/src/layout/Main/MainRoutes.tsx
import { Navigate, Route, Routes } from "react-router-dom";

import BillingPage from "../../pages/BillingPage";
import ContractsPage from "../../pages/ContractsPage";
import GasPage from "../../pages/GasPage";
import InquiryDetailPage from "../../pages/InquiryDetailPage";
import InquiriesPage from "../../pages/InquiriesPage";
import ReportsPage from "../../pages/ReportsPage";

export default function MainRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/inquiries" replace />} />
      <Route path="/inquiries" element={<InquiriesPage />} />
      <Route path="/inquiries/:inquiryId" element={<InquiryDetailPage />} />
      <Route path="/gas" element={<GasPage />} />
      <Route path="/contracts" element={<ContractsPage />} />
      <Route path="/reports" element={<ReportsPage />} />
      <Route path="/billing" element={<BillingPage />} />
      <Route path="*" element={<Navigate to="/inquiries" replace />} />
    </Routes>
  );
}