// frontend/console/shell/src/app/App.tsx

import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";

import {
  AuthProvider,
  useAuthContext,
} from "../auth/application/AuthContext";
import AuthPage from "../pages/AuthPage";
import InvitationPage from "../pages/InvitationPage";
import MainPage from "../pages/MainPage";

function InvitationRoute() {
  const location = useLocation();

  const searchParams =
    new URLSearchParams(location.search);

  const token =
    searchParams.get("token");

  if (!token) {
    return (
      <Navigate
        to="/"
        replace
      />
    );
  }

  return <InvitationPage />;
}

function RootContent() {
  const {
    user,
    loading,
    currentMember,
    loadingMember,
    memberError,
  } = useAuthContext();

  const location = useLocation();

  if (loading) {
    return (
      <div style={{ padding: 24 }}>
        認証状態を確認しています...
      </div>
    );
  }

  return (
    <Routes>
      <Route
        path="/invitation"
        element={<InvitationRoute />}
      />

      <Route
        path="/*"
        element={
          !user ? (
            <AuthPage />
          ) : loadingMember ? (
            <div style={{ padding: 24 }}>
              会社情報を準備しています...
            </div>
          ) : memberError ? (
            <div style={{ padding: 24 }}>
              {memberError}
            </div>
          ) : !currentMember?.companyId ? (
            <div style={{ padding: 24 }}>
              会社情報を確認しています...
            </div>
          ) : (
            // 会社情報の取得完了後にのみMainPageを構築する
            // ルート遷移のたびにMainPage/Headerを必ず構築し直す
            <MainPage
              key={location.key}
            />
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