//frontend\admin\shell\src\App.tsx
import { useState } from "react";

import {
  clearAdminSession,
  hasAdminSession,
} from "./auth/adminAuth";

import LoginPage from "./pages/LoginPage";
import MainPage from "./pages/MainPage";

export default function App() {
  const [
    authenticated,
    setAuthenticated,
  ] = useState(
    () => hasAdminSession(),
  );

  function handleLogin() {
    setAuthenticated(true);
  }

  function handleLogout() {
    clearAdminSession();
    setAuthenticated(false);
  }

  if (!authenticated) {
    return (
      <LoginPage
        onLogin={handleLogin}
      />
    );
  }

  return (
    <MainPage
      onLogout={handleLogout}
    />
  );
}