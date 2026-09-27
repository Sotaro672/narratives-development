// frontend/mall/src/pages/ChatListPage.tsx

import Layout from "../components/layout/Layout";
import ChatListPane from "../features/inquiry/presentation/components/ChatListPane";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-list-page.css";

export default function ChatListPage() {
  return (
    <Layout
      title="AMOL"
      showFooter
      mode="mypage"
      mainClassName="chat-list-page-layout"
    >
      <ChatListPane />
    </Layout>
  );
}