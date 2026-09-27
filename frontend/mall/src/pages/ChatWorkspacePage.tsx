// frontend/mall/src/pages/ChatWorkspacePage.tsx

import type { CSSProperties } from "react";
import { Outlet, useLocation } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import Layout from "../components/layout/Layout";
import StatePanel from "../components/ui/StatePanel";
import ChatListPane from "../features/inquiry/presentation/components/ChatListPane";

import ChatListPage from "./ChatListPage";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-list-page.css";

const workspaceStyle: CSSProperties = {
  display: "grid",
  gridTemplateColumns: "minmax(320px, 380px) minmax(0, 1fr)",
  width: "100%",
  height: "calc(100dvh - var(--amol-header-height))",
  minHeight: 0,
  overflow: "hidden",
  background: "#f8fafc",
};

const listPaneStyle: CSSProperties = {
  minWidth: 0,
  minHeight: 0,
  height: "100%",
  overflowX: "hidden",
  overflowY: "auto",
  borderRight: "1px solid rgba(15, 23, 42, 0.1)",
  background: "#ffffff",
  boxSizing: "border-box",
};

const detailPaneStyle: CSSProperties = {
  minWidth: 0,
  minHeight: 0,
  height: "100%",
  overflow: "hidden",
  background: "#f8fafc",
};

const emptyPaneStyle: CSSProperties = {
  display: "grid",
  width: "100%",
  height: "100%",
  minHeight: 0,
  placeItems: "center",
  padding: 24,
  boxSizing: "border-box",
};

export default function ChatWorkspacePage() {
  const location = useLocation();
  const isMobilePortrait = useMobilePortrait();
  const isChatListRoute =
    location.pathname === "/chats" ||
    location.pathname === "/chats/";

  if (isMobilePortrait) {
    if (isChatListRoute) {
      return <ChatListPage />;
    }

    return <Outlet />;
  }

  return (
    <Layout
      title="AMOL"
      showFooter={false}
      mode="mypage"
      mainClassName="chat-workspace-page-layout"
      disableFooterPaddingOnDesktop
    >
      <div
        className="chat-workspace-page"
        style={workspaceStyle}
      >
        <aside
          className="chat-workspace-page__list"
          style={listPaneStyle}
          aria-label="チャット一覧"
        >
          <ChatListPane />
        </aside>

        <section
          className="chat-workspace-page__detail"
          style={detailPaneStyle}
          aria-label="チャット詳細"
        >
          {isChatListRoute ? (
            <div
              className="chat-workspace-page__empty"
              style={emptyPaneStyle}
            >
              <StatePanel
                variant="empty"
                title="チャットを選択してください。"
              />
            </div>
          ) : (
            <Outlet context={{ embedded: true }} />
          )}
        </section>
      </div>
    </Layout>
  );
}