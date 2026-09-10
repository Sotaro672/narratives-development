// frontend/admin/shell/src/layout/Main/MainRoutes.tsx

import { Navigate, Route, Routes } from "react-router-dom";

import AvatarPage from "../../pages/AvatarPage";
import BillingPage from "../../pages/BillingPage";
import ContractDetailPage from "../../pages/ContractDetailPage";
import ContractsPage from "../../pages/ContractsPage";
import GasPage from "../../pages/GasPage";
import InquiryDetailPage from "../../pages/InquiryDetailPage";
import InquiriesPage from "../../pages/InquiriesPage";
import ListDetailPage from "../../pages/ListDetailPage";
import MintDetailPage from "../../pages/MintDetailPage";
import NewsCreatePage from "../../pages/NewsCreatePage";
import NewsDetailPage from "../../pages/NewsDetailPage";
import NewsPage from "../../pages/NewsPage";
import ProductBlueprintDetailPage from "../../pages/ProductBlueprintDetailPage";
import ReportDetailPage from "../../pages/ReportDetailPage";
import ReportsPage from "../../pages/ReportsPage";
import TokenBlueprintDetailPage from "../../pages/TokenBlueprintDetailPage";

export default function MainRoutes() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/inquiries" replace />} />
      <Route path="/inquiries" element={<InquiriesPage />} />
      <Route path="/inquiries/:inquiryId" element={<InquiryDetailPage />} />
      <Route path="/gas" element={<GasPage />} />
      <Route path="/gas/mints/:mintId" element={<MintDetailPage />} />
      <Route path="/news" element={<NewsPage />} />
      <Route path="/news/create" element={<NewsCreatePage />} />
      <Route path="/news/:newsId" element={<NewsDetailPage />} />
      <Route path="/contracts" element={<ContractsPage />} />
      <Route path="/contracts/:companyId" element={<ContractDetailPage />} />
      <Route path="/contracts/:companyId/lists/:listId" element={<ListDetailPage />} />
      <Route path="/contracts/:companyId/token-blueprints/:tokenBlueprintId" element={<TokenBlueprintDetailPage />} />
      <Route path="/contracts/:companyId/product-blueprints/:productBlueprintId" element={<ProductBlueprintDetailPage />} />
      <Route path="/avatars" element={<AvatarPage />} />
      <Route path="/reports" element={<ReportsPage />} />
      <Route path="/reports/:reportId" element={<ReportDetailPage />} />
      <Route path="/billing" element={<BillingPage />} />
      <Route path="*" element={<Navigate to="/inquiries" replace />} />
    </Routes>
  );
}