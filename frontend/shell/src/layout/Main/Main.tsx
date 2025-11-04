//frontend\shell\src\layout\Main\Main.tsx
import { Routes, Route } from "react-router-dom";
import InquiryManagementPage from "../../../../inquiry/src/pages/InquiryManagementPage";
import "./Main.css";

export default function Main() {
  return (
    <div className="main-content">
      <Routes>
        {/* Sidebarの「問い合わせ」を押した時に表示されるページ */}
        {<Route path="/inquiry" element={<InquiryManagementPage />} />}
      </Routes>
      <h1 className="text-2xl font-semibold">問い合わせ管理</h1>
      {/* 必要に応じて他のページを追加 */}
      {/* <Route path="/listings" element={<ListingsPage />} /> */}
    </div>
  );
}
