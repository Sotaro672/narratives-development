import { Routes, Route, Navigate } from "react-router-dom";
import AnnounceDashboardPage from "./pages/AnnounceDashboardPage";
import AnnounceDetailPage from "./pages/AnnounceDetailPage";
import AnnounceEditorPage from "./pages/AnnounceEditorPage";
import AnnounceHistoryPage from "./pages/AnnounceHistoryPage";

/**
 * AnnounceRoutes
 * ------------------------------------------------------------------------
 * アナウンス（告知・お知らせ）モジュールのルーティング構成。
 * shell から import("announce/routes") でロードされる。
 *
 * - Dashboard: 一覧・配信状況を表示
 * - Editor: 新規投稿・編集
 * - History: 過去のアナウンス履歴
 * - Detail: 個別アナウンス詳細
 * ------------------------------------------------------------------------
 */
export default function AnnounceRoutes() {
  return (
    <Routes>
      {/* ─────────────── Dashboard（一覧）─────────────── */}
      <Route path="/" element={<AnnounceDashboardPage />} />

      {/* ─────────────── 新規投稿・編集 ─────────────── */}
      <Route path="/editor" element={<AnnounceEditorPage />} />
      <Route path="/editor/:id" element={<AnnounceEditorPage />} />

      {/* ─────────────── 履歴・詳細 ─────────────── */}
      <Route path="/history" element={<AnnounceHistoryPage />} />
      <Route path="/:id" element={<AnnounceDetailPage />} />

      {/* ─────────────── その他（未定義ルート）─────────────── */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
