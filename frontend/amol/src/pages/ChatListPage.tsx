// frontend/amol/src/pages/ChatListPage.tsx

import Layout from "../components/layout/Layout";

import InquiryListItem from "../features/inquiry/presentation/components/InquiryListItem";
import { useInquiryListPage } from "../features/inquiry/presentation/hooks/useInquiryListPage";

import "../styles/page-layout.css";
import "../features/inquiry/presentation/styles/inquiry-list-page.css";

export default function ChatListPage() {
  const {
    sortedItems,
    loading,
    navigatingId,
    error,
    handleOpenChat,
  } = useInquiryListPage();

  return (
    <Layout
      title="チャット"
      showBackButton
      showFooter
      mode="mypage"
      mainClassName="chat-list-page-layout"
    >
      <section className="page-section content-page-section chat-list-page">
        {error ? (
          <div
            className="chat-list-page__error"
            role="alert"
          >
            {error}
          </div>
        ) : null}

        {loading ? (
          <div className="chat-list-page__state">
            読み込み中...
          </div>
        ) : null}

        {!loading && sortedItems.length === 0 ? (
          <div className="chat-list-page__empty">
            現在、チャットはありません。
          </div>
        ) : null}

        {!loading && sortedItems.length > 0 ? (
          <div
            className="chat-list-page__list"
            aria-label="チャット一覧"
          >
            {sortedItems.map((item) => (
              <InquiryListItem
                key={`inquiry:${item.id}`}
                item={item}
                navigating={navigatingId === item.id}
                onOpen={() => {
                  void handleOpenChat(item);
                }}
              />
            ))}
          </div>
        ) : null}
      </section>
    </Layout>
  );
}