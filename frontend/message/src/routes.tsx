import { Routes, Route, Navigate } from "react-router-dom";
import MessageDashboardPage from "./pages/MessageDashboardPage";
import MessageThreadPage from "./pages/MessageThreadPage";
import MessageDetailPage from "./pages/MessageDetailPage";
import MessageSettingsPage from "./pages/MessageSettingsPage";

/**
 * MessageRoutes
 * メッセージ・通知モジュールのルーティング構成。
 * shell から import("message/routes") でロードされる。
 */
export default function MessageRoutes() {
  return (
    <Routes>
      <Route path="/" element={<MessageDashboardPage />} />
      <Route path="/thread/:id" element={<MessageThreadPage />} />
      <Route path="/detail/:id" element={<MessageDetailPage />} />
      <Route path="/settings" element={<MessageSettingsPage />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
