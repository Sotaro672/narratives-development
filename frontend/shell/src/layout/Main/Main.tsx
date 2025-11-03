import { Routes, Route } from "react-router-dom";
import InquiryManagementPage from "../../../../inquiry/src/pages/InquiryManagementPage";
import "./Main.css";

export default function Main() {
  return (
    <div className="main-content">
      <Routes>
        {/* Sidebarの「問い合わせ」を押した時に表示されるページ */}
        <Route path="/inquiry" element={<InquiryManagementPage />} />

        {/* 必要に応じて他のページを追加 */}
        {/* <Route path="/listings" element={<ListingsPage />} /> */}
      </Routes>
    </div>
  );
}
