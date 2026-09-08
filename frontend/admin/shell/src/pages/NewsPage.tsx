// frontend/admin/shell/src/pages/NewsPage.tsx

import { useMemo, useState } from "react";
import type { FormEvent } from "react";

import { useNews } from "../features/news/hooks/useNews";
import type { News } from "../shared/type/news";
import Button from "../shared/ui/Button/Button";
import Page, { PageHeader } from "../shared/ui/Page/Page";
import RefreshButton from "../shared/ui/RefreshButton/RefreshButton";
import Table, { type TableColumn } from "../shared/ui/Table/Table";
import { formatDateTime } from "../shared/util/dateFormat";

function getBodySummary(body: string): string {
  const value = body.trim();
  if (!value) return "-";
  return value.length > 120 ? `${value.slice(0, 120)}…` : value;
}

export default function NewsPage() {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");

  const {
    items,
    loading,
    error,
    publishing,
    publishError,
    page,
    totalCount,
    totalPages,
    hasPreviousPage,
    hasNextPage,
    setPage,
    publish,
    reload,
  } = useNews();

  const canPublish =
    title.trim().length > 0 &&
    body.trim().length > 0 &&
    !publishing;

  const columns = useMemo<TableColumn<News>[]>(
    () => [
      {
        key: "publishedAt",
        header: "公開日時",
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
        key: "createdAt",
        header: "作成日時",
        render: (news) => formatDateTime(news.createdAt),
        sortValue: (news) => new Date(news.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const normalizedTitle = title.trim();
    const normalizedBody = body.trim();

    if (!normalizedTitle || !normalizedBody || publishing) {
      return;
    }

    const confirmed = window.confirm(
      "ConsoleとMallの全ユーザーへこの通知を公開します。公開後は編集・削除できません。通知しますか？",
    );

    if (!confirmed) {
      return;
    }

    const created = await publish({
      title: normalizedTitle,
      body: normalizedBody,
    });

    if (!created) {
      return;
    }

    setTitle("");
    setBody("");
  };

  return (
    <Page>
      <PageHeader
        title="通知"
        actions={
          <RefreshButton
            onClick={reload}
            loading={loading}
            title="リフレッシュ"
            ariaLabel="通知一覧をリフレッシュ"
          />
        }
      />

      <section>
        <h2>システム通知</h2>
        <p>ConsoleとMallの全ユーザーへシステムに関するお知らせを通知します。</p>

        <form onSubmit={handleSubmit}>
          <div>
            <label htmlFor="news-title">タイトル</label>
            <input
              id="news-title"
              type="text"
              value={title}
              disabled={publishing}
              autoComplete="off"
              onChange={(event) => setTitle(event.target.value)}
            />
          </div>

          <div>
            <label htmlFor="news-body">本文</label>
            <textarea
              id="news-body"
              value={body}
              disabled={publishing}
              rows={8}
              onChange={(event) => setBody(event.target.value)}
            />
          </div>

          {publishError ? (
            <p role="alert">
              通知に失敗しました。{publishError}
            </p>
          ) : null}

          <Button
            type="submit"
            variant="primary"
            loading={publishing}
            disabled={!canPublish}
          >
            {publishing ? "通知中..." : "通知する"}
          </Button>
        </form>
      </section>

      <section>
        <h2>通知履歴</h2>

        {loading && items.length === 0 ? (
          <p>通知履歴を読み込んでいます。</p>
        ) : null}

        {!loading && error ? (
          <p role="alert">
            通知履歴の取得に失敗しました。{error}
          </p>
        ) : null}

        {!error && (items.length > 0 || !loading) ? (
          <>
            <p>{totalCount}件</p>

            <Table
              columns={columns}
              rows={items}
              getRowKey={(news) => news.id}
              emptyMessage="通知履歴はありません。"
              filteredEmptyMessage="条件に一致する通知はありません。"
            />

            {totalPages > 1 ? (
              <nav aria-label="通知履歴のページ送り">
                <Button
                  size="sm"
                  variant="secondary"
                  disabled={!hasPreviousPage || loading}
                  onClick={() => setPage(page - 1)}
                >
                  前へ
                </Button>

                <span>
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