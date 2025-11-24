// frontend/console/shell/src/app/App.tsx
import {
  BrowserRouter,
  Routes,
  Route,
  Navigate,
  useLocation,
} from "react-router-dom";
import MainPage from "../pages/MainPage";
import AuthPage from "../auth/presentation/pages/AuthPage";
import InvitationPage from "../auth/presentation/pages/InvitationPage";
import { AuthProvider } from "../auth/application/AuthContext";
import { useAuth } from "../auth/presentation/hook/useCurrentMember";

/**
 * 招待ページ専用のルートガード
 * - URL に ?token=xxx が付いている場合のみ InvitationPage を表示
 * - それ以外（直接アクセスなど）は "/" にリダイレクト
 */
function InvitationRoute() {
  const location = useLocation();
  const searchParams = new URLSearchParams(location.search);
  const token = searchParams.get("token");

  if (!token) {
    // token が無ければ招待ページには入れない
    // "/" に戻し、RootContent 側のルーティングで Auth / Main を切り替える
    return <Navigate to="/" replace />;
  }

  return <InvitationPage />;
}

function RootContent() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div style={{ padding: 24 }}>認証状態を確認しています...</div>;
  }

  return (
    <Routes>
      {/* ★ 招待ページ: token 付き URL からのみアクセス可能 */}
      <Route path="/invitation" element={<InvitationRoute />} />

      {/* ★ それ以外のルート:
          - 未ログイン: AuthPage
          - ログイン済み: MainPage */}
      <Route path="/*" element={user ? <MainPage /> : <AuthPage />} />
    </Routes>
  );
}

/**
 * App.tsx
 * - /invitation は InvitationRoute で token チェック
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
