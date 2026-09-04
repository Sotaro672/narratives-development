// frontend/admin/shell/src/App.tsx
import { useEffect, useState } from "react";

import {
  observeAdminAuth,
  signOutAdmin,
} from "./auth/application/adminAuth";

import LoginPage from "./pages/LoginPage";
import MainPage from "./pages/MainPage";

export default function App() {
  const [authenticated, setAuthenticated] = useState(false);
  const [authReady, setAuthReady] = useState(false);

  useEffect(() => {
    const unsubscribe = observeAdminAuth((user) => {
      setAuthenticated(user !== null);
      setAuthReady(true);
    });

    return unsubscribe;
  }, []);

  function handleLogin() {
    setAuthenticated(true);
  }

  async function handleLogout() {
    try {
      await signOutAdmin();
    } catch (error) {
      console.error("[admin-auth] sign out failed", error);
    } finally {
      setAuthenticated(false);
    }
  }

  if (!authReady) {
    return null;
  }

  if (!authenticated) {
    return <LoginPage onLogin={handleLogin} />;
  }

  return <MainPage onLogout={handleLogout} />;
}