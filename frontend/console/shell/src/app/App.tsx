// frontend/console/shell/src/app/App.tsx
import { BrowserRouter, Routes, Route } from "react-router-dom";
import MainPage from "../pages/MainPage";
import AuthPage from "../auth/presentation/pages/AuthPage";
import InvitationPage from "../auth/presentation/pages/InvitationPage";
import { AuthProvider } from "../auth/application/AuthContext";
import { useAuth } from "../auth/presentation/hook/useCurrentMember";

function RootContent() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div style={{ padding: 24 }}>認証状態を確認しています...</div>;
  }

  return (
    <Routes>
      {/* ★ 招待ページ: MainPage 配下ではなく独立したページ */}
      <Route path="/invitation" element={<InvitationPage />} />

      {/* ★ それ以外のルート:
          - 未ログイン: AuthPage
          - ログイン済み: MainPage */}
      <Route
        path="/*"
        element={user ? <MainPage /> : <AuthPage />}
      />
    </Routes>
  );
}

/**
 * App.tsx
 * - /invitation は InvitationPage を直接表示
 * - それ以外は Firebase Auth の状態に応じて AuthPage / MainPage を切り替える
 */
export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <RootContent />
      </AuthProvider>
    </BrowserRouter>
  );
}
