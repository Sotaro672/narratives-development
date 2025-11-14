// src/app/App.tsx
import * as React from "react";
import { BrowserRouter } from "react-router-dom";
import MainPage from "../pages/MainPage";
import AuthPage from "../auth/pages/AuthPage";
import { AuthProvider } from "../auth/application/AuthContext";
import { useAuth } from "../auth/application/useAuth";

function RootContent() {
  const { user, loading } = useAuth();

  if (loading) {
    return <div style={{ padding: 24 }}>認証状態を確認しています...</div>;
  }

  // 未ログインなら AuthPage
  if (!user) {
    return <AuthPage />;
  }

  // ログイン済みなら Console メイン画面
  return <MainPage />;
}

/**
 * App.tsx
 * - Firebase Auth の状態に応じて AuthPage / MainPage を切り替える
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
