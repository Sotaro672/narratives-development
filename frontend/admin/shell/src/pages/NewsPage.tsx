// frontend/admin/shell/src/pages/NewsPage.tsx

import { useMemo } from "react";
import { useNavigate } from "react-router-dom";

import { useNews } from "../features/news/presentation/hooks/useNews";
import type { News } from "../shared/type/news";
import Button from "../shared/ui/Button/Button";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
import Table, { type TableColumn } from "../shared/ui/Table/Table";
import { formatDateTime } from "../shared/util/dateFormat";

import "./NewsPage.css";

function getBodySummary(body: string): string {
  if (!body) {
    return "-";
  }

  return body.length > 120 ? `${body.slice(0, 120)}…` : body;
}

export default function NewsPage() {
  const navigate = useNavigate();

  const {
    items,
    loading,
    error,
    page,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    setPage,
    reload,
  } = useNews();

  const columns = useMemo<TableColumn<News>[]>(
    () => [
      {
        key: "publishedAt",
        header: "配信日時",
        render: (news) => formatDateTime(news.publishedAt),
        sortValue: (news) => new Date(news.publishedAt).getTime(),
        nowrap: true,
      },
      {
        key: "title",
        header: "タイトル",
        render: (news) => news.title,
        filter: {
          getValue: (news) => news.title,
          placeholder: "タイトルで絞り込み",
        },
        minWidth: "220px",
      },
      {
        key: "body",
        header: "本文",
        render: (news) => getBodySummary(news.body),
        filter: {
          getValue: (news) => news.body,
          placeholder: "本文で絞り込み",
        },
        minWidth: "360px",
      },
      {
        key: "createdByName",
        header: "作成者",
        render: (news) => news.createdByName || "-",
        filter: {
          getValue: (news) => news.createdByName || "",
          placeholder: "作成者で絞り込み",
        },
        minWidth: "160px",
      },
      {
        key: "createdAt",
        header: "作成日時",
        render: (news) => formatDateTime(news.createdAt),
        sortValue: (news) => new Date(news.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Page>
      <PageHeader
        title="通知"
        actions={
          <>
            <Button type="button" variant="primary" onClick={() => navigate("/news/create")}>
              通知を作成
            </Button>

            <RefreshButton
              onClick={reload}
              loading={loading}
              title="リフレッシュ"
              ariaLabel="通知一覧をリフレッシュ"
            />
          </>
        }
      />

      <section className="news-page__history-section">
        <div className="news-page__section-header news-page__section-header--history">
          <div>
            <h2 className="news-page__section-title">配信済み通知</h2>
            <p className="news-page__section-description">
              これまでConsoleとMallへ配信したシステム通知の一覧です。
            </p>
          </div>

          {!error && !loading ? <span className="news-page__count">{totalCount}件</span> : null}
        </div>

        {loading && items.length === 0 ? (
          <p className="news-page__state">配信済み通知を読み込んでいます。</p>
        ) : null}

        {!loading && error ? (
          <p className="news-page__error" role="alert">
            配信済み通知の取得に失敗しました。{error}
          </p>
        ) : null}

        {!error && (items.length > 0 || !loading) ? (
          <>
            <Table
              columns={columns}
              rows={items}
              getRowKey={(news) => news.id}
              onRowClick={(news) =>
                navigate(`/news/${encodeURIComponent(news.id)}`, {
                  state: { news },
                })
              }
              emptyMessage="配信済み通知はありません。"
              filteredEmptyMessage="条件に一致する通知はありません。"
            />

            {totalPages > 1 ? (
              <nav className="news-page__pagination" aria-label="配信済み通知のページ送り">
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={!hasPreviousPage || loading}
                  onClick={() => setPage(page - 1)}
                >
                  前へ
                </Button>

                <span className="news-page__page-label">
                  {page} / {totalPages}
                </span>

                <Button
                  size="sm"
                  variant="secondary"
                  disabled={!hasNextPage || loading}
                  onClick={() => setPage(page + 1)}
                >
                  次へ
                </Button>
              </nav>
            ) : null}
          </>
        ) : null}
      </section>
    </Page>
  );
}