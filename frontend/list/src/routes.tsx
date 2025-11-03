import { Routes, Route, Navigate } from "react-router-dom";
import ListingListPage from "./pages/ListingListPage";
import ListingDetailPage from "./pages/ListingDetailPage";
import ListingCreatePage from "./pages/ListingCreatePage";

/**
 * ListingsRoutes
 * 出品モジュールのルート構成。
 * shell から import("listings/routes") でロードされる。
 */
export default function ListingsRoutes() {
  return (
    <Routes>
      <Route path="/" element={<ListingListPage />} />
      <Route path="/create" element={<ListingCreatePage />} />
      <Route path="/:id" element={<ListingDetailPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
