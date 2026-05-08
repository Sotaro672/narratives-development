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

function InvitationRoute() {
  const location = useLocation();
  const searchParams = new URLSearchParams(location.search);
  const token = searchParams.get("token");
  if (!token) return <Navigate to="/" replace />;
  return <InvitationPage />;
}

function RootContent() {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <div style={{ padding: 24 }}>認証状態を確認しています...</div>;
  }

  return (
    <Routes>
      <Route path="/invitation" element={<InvitationRoute />} />
      <Route
        path="/*"
        element={
          user ? (
            // ルート遷移のたびに MainPage/Header を必ず build し直す
            <MainPage key={location.key} />
          ) : (
            <AuthPage />
          )
        }
      />
    </Routes>
  );
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <RootContent />
      </AuthProvider>
    </BrowserRouter>
  );
}