// frontend/console/shell/src/app/App.tsx
import { BrowserRouter, Routes, Route } from "react-router-dom";
import MainPage from "../pages/MainPage";
import AuthPage from "../auth/presentation/pages/AuthPage";
import { AuthProvider } from "../auth/application/AuthContext";
import { useAuth } from "../auth/presentation/hook/useCurrentMember";
import InvitationPage from "../auth/presentation/pages/InvitationPage";

function RootContent() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div style={{ padding: 24 }}>認証状態を確認しています...</div>;
  }

  return (
    <Routes>
      {/* メール内リンク用：/invitation に遷移したら InvitationPage を表示 */}
      <Route path="/invitation" element={<InvitationPage />} />

      {/* それ以外のパスは従来どおり AuthPage / MainPage を切り替え */}
      <Route path="/*" element={user ? <MainPage /> : <AuthPage />} />
    </Routes>
  );
}

/**
 * App.tsx
 * - Firebase Auth の状態に応じて AuthPage / MainPage を切り替える
 * - 追加で /invitation ルートを用意
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
