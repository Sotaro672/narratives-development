// frontend/amol/src/pages/AvatarReviewPage.tsx

import {
  useCallback,
  useEffect,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import { getPublicAvatar } from "../features/avatar/api/avatarApi";
import {
  fetchAvatarReviews,
  type AvatarReviewPageResponse,
} from "../features/avatar-review/api/avatarReviewApi";

import "../styles/page-layout.css";
import "../styles/avatar-review-page.css";

const PER_PAGE = 20;

function formatCreatedAt(
  value: string,
): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (
    Number.isNaN(date.getTime())
  ) {
    return "";
  }

  return new Intl.DateTimeFormat(
    "ja-JP",
    {
      year: "numeric",
      month: "numeric",
      day: "numeric",
    },
  ).format(date);
}

export default function AvatarReviewPage() {
  const navigate = useNavigate();

  const { avatarId = "" } =
    useParams<{
      avatarId: string;
    }>();

  const [avatarName, setAvatarName] =
    useState("");
  const [result, setResult] =
    useState<AvatarReviewPageResponse | null>(
      null,
    );
  const [page, setPage] =
    useState(1);
  const [loading, setLoading] =
    useState(true);
  const [error, setError] =
    useState<string | null>(null);

  const load = useCallback(
    async (nextPage: number) => {
      const id =
        avatarId.trim();

      if (!id) {
        setError(
          "アバターIDが指定されていません。",
        );
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const [
          avatar,
          reviews,
        ] = await Promise.all([
          getPublicAvatar({
            avatarId: id,
          }),
          fetchAvatarReviews({
            avatarId: id,
            page: nextPage,
            perPage: PER_PAGE,
          }),
        ]);

        setAvatarName(
          avatar?.avatarName ??
            "",
        );
        setResult(reviews);
        setPage(nextPage);
      } catch (caught) {
        setError(
          caught instanceof Error
            ? caught.message
            : "評価の取得に失敗しました。",
        );
      } finally {
        setLoading(false);
      }
    },
    [avatarId],
  );

  useEffect(() => {
    void load(1);
  }, [load]);

  return (
    <Layout
      title={
        avatarName
          ? `${avatarName}の評価`
          : "評価"
      }
      mode="mypage"
      showBackButton
      onBackButtonClick={() =>
        navigate(-1)
      }
    >
      <section className="content-page-section avatar-review-page">
        {loading ? (
          <p className="avatar-review-page__message">
            読み込み中です...
          </p>
        ) : null}

        {!loading && error ? (
          <div
            role="alert"
            className="avatar-review-page__message"
          >
            {error}
          </div>
        ) : null}

        {!loading &&
        !error &&
        result ? (
          <>
            <header className="avatar-review-page__header">
              <h1 className="avatar-review-page__title">
                {avatarName ||
                  "アバター"}
                の評価
              </h1>

              <div className="avatar-review-page__summary">
                <div className="avatar-review-page__summary-item">
                  <span>
                    良かった
                  </span>
                  <strong>
                    {
                      result.goodCount
                    }
                  </strong>
                </div>

                <div className="avatar-review-page__summary-item">
                  <span>
                    残念だった
                  </span>
                  <strong>
                    {
                      result.disappointedCount
                    }
                  </strong>
                </div>
              </div>
            </header>

            {result.items.length ===
            0 ? (
              <p className="avatar-review-page__empty">
                まだ評価はありません。
              </p>
            ) : (
              <div className="avatar-review-page__list">
                {result.items.map(
                  (review) => (
                    <article
                      key={
                        review.id
                      }
                      className="avatar-review-card"
                    >
                      <div className="avatar-review-card__header">
                        <strong
                          className={
                            review.evaluation ===
                            "good"
                              ? "avatar-review-card__evaluation avatar-review-card__evaluation--good"
                              : "avatar-review-card__evaluation avatar-review-card__evaluation--disappointed"
                          }
                        >
                          {review.evaluation ===
                          "good"
                            ? "良かった"
                            : "残念だった"}
                        </strong>

                        <time className="avatar-review-card__date">
                          {formatCreatedAt(
                            review.createdAt,
                          )}
                        </time>
                      </div>

                      <p className="avatar-review-card__comment">
                        {
                          review.comment
                        }
                      </p>
                    </article>
                  ),
                )}
              </div>
            )}

            <div className="avatar-review-page__pagination">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() => {
                  void load(
                    page - 1,
                  );
                }}
              >
                前へ
              </button>

              <span>
                {page}
              </span>

              <button
                type="button"
                disabled={
                  !result.hasNext
                }
                onClick={() => {
                  void load(
                    page + 1,
                  );
                }}
              >
                次へ
              </button>
            </div>
          </>
        ) : null}
      </section>
    </Layout>
  );
}