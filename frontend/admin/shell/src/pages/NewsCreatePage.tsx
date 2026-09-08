// frontend/admin/shell/src/pages/NewsCreatePage.tsx

import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate } from "react-router-dom";

import { useNews } from "../features/news/hooks/useNews";
import Button from "../shared/ui/Button/Button";
import Page, { PageHeader } from "../shared/ui/Page/Page";

import "./NewsCreatePage.css";

const NEWS_CREATE_FORM_ID = "news-create-form";

export default function NewsCreatePage() {
  const navigate = useNavigate();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");

  const {
    publishing,
    publishError,
    publish,
  } = useNews();

  const canPublish =
    title.trim().length > 0 &&
    body.trim().length > 0 &&
    !publishing;

  const handleSubmit = async (
    event: FormEvent<HTMLFormElement>,
  ) => {
    event.preventDefault();

    const normalizedTitle = title.trim();
    const normalizedBody = body.trim();

    if (
      !normalizedTitle ||
      !normalizedBody ||
      publishing
    ) {
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

    navigate("/news", {
      replace: true,
    });
  };

  return (
    <Page>
      <PageHeader
        title="通知を作成"
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="通知一覧へ戻る"
            onClick={() => navigate("/news")}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
        actions={
          <Button
            type="submit"
            form={NEWS_CREATE_FORM_ID}
            variant="primary"
            loading={publishing}
            disabled={!canPublish}
          >
            {publishing
              ? "通知中..."
              : "通知する"}
          </Button>
        }
      />

      <section className="news-create-page__section">
        <div className="news-create-page__section-header">
          <h2 className="news-create-page__section-title">
            システム通知
          </h2>

          <p className="news-create-page__section-description">
            ConsoleとMallの全ユーザーへシステムに関するお知らせを通知します。
          </p>
        </div>

        <form
          id={NEWS_CREATE_FORM_ID}
          className="news-create-page__form"
          onSubmit={handleSubmit}
        >
          <div className="news-create-page__field">
            <label
              className="news-create-page__label"
              htmlFor="news-title"
            >
              タイトル
            </label>

            <input
              id="news-title"
              className="news-create-page__input"
              type="text"
              value={title}
              disabled={publishing}
              autoComplete="off"
              onChange={(event) =>
                setTitle(event.target.value)
              }
            />
          </div>

          <div className="news-create-page__field">
            <label
              className="news-create-page__label"
              htmlFor="news-body"
            >
              本文
            </label>

            <textarea
              id="news-body"
              className="news-create-page__textarea"
              value={body}
              disabled={publishing}
              rows={8}
              onChange={(event) =>
                setBody(event.target.value)
              }
            />
          </div>

          {publishError ? (
            <p
              className="news-create-page__error"
              role="alert"
            >
              通知に失敗しました。{publishError}
            </p>
          ) : null}
        </form>
      </section>
    </Page>
  );
}