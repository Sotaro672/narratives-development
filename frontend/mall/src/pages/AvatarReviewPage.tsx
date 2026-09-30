// frontend/mall/src/pages/AvatarReviewPage.tsx

import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Card from "../components/ui/Card";
import Pagination from "../components/ui/Pagination";
import TextState from "../components/ui/TextState";
import {
  fetchAvatarReviews,
  type AvatarReviewEvaluation,
  type AvatarReviewPageResponse,
} from "../features/avatar-review/api/avatarReviewApi";
import { getPublicAvatar } from "../features/avatar/api/avatarApi";

import "../styles/page-layout.css";
import "../styles/avatar-review-page.css";

const PER_PAGE = 20;

function formatCreatedAt(value: string): string {
  if (!value) {
    return "";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return "";
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "numeric",
    day: "numeric",
  }).format(date);
}

export default function AvatarReviewPage() {
  const { avatarId = "" } = useParams<{ avatarId: string }>();

  const [avatarName, setAvatarName] = useState("");
  const [result, setResult] = useState<AvatarReviewPageResponse | null>(null);
  const [evaluationFilter, setEvaluationFilter] = useState<AvatarReviewEvaluation | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (
      nextPage: number,
      evaluation: AvatarReviewEvaluation | null = evaluationFilter,
    ) => {
      const id = avatarId.trim();

      if (!id) {
        setError("アバターIDが指定されていません。");
        setLoading(false);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        const [avatar, reviews] = await Promise.all([
          getPublicAvatar({ avatarId: id }),
          fetchAvatarReviews({
            avatarId: id,
            page: nextPage,
            perPage: PER_PAGE,
            evaluation: evaluation ?? undefined,
          }),
        ]);

        setAvatarName(avatar?.avatarName ?? "");
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
    [avatarId, evaluationFilter],
  );

  useEffect(() => {
    void load(1);
  }, [load]);

  const handleFilterChange = (evaluation: AvatarReviewEvaluation) => {
    const nextEvaluation =
      evaluationFilter === evaluation ? null : evaluation;

    setEvaluationFilter(nextEvaluation);
  };

  const totalPages = result
    ? Math.max(1, Math.ceil(result.total / result.perPage))
    : 1;

  const emptyMessage =
    evaluationFilter === "good"
      ? "良かった評価はまだありません。"
      : evaluationFilter === "disappointed"
        ? "残念だった評価はまだありません。"
        : "まだ評価はありません。";

  return (
    <Layout title="AMOL" mode="mypage">
      <section className="content-page-section avatar-review-page">
        {loading ? (
          <TextState variant="loading" className="avatar-review-page__message">
            読み込み中です...
          </TextState>
        ) : null}

        {!loading && error ? (
          <TextState variant="error" className="avatar-review-page__message">
            {error}
          </TextState>
        ) : null}

        {!loading && !error && result ? (
          <>
            <header className="avatar-review-page__header">
              <h1 className="avatar-review-page__title">
                {avatarName || "アバター"}の評価
              </h1>

              <div className="avatar-review-page__summary">
                <Card
                  variant="panel"
                  interactive
                  highlighted={evaluationFilter === "good"}
                  className="avatar-review-page__summary-item"
                  aria-pressed={evaluationFilter === "good"}
                  aria-label={`良かった ${result.goodCount}件で絞り込む`}
                  onClick={() => {
                    handleFilterChange("good");
                  }}
                >
                  <span>良かった</span>
                  <strong>{result.goodCount}</strong>
                </Card>

                <Card
                  variant="panel"
                  interactive
                  highlighted={evaluationFilter === "disappointed"}
                  className="avatar-review-page__summary-item"
                  aria-pressed={evaluationFilter === "disappointed"}
                  aria-label={`残念だった ${result.disappointedCount}件で絞り込む`}
                  onClick={() => {
                    handleFilterChange("disappointed");
                  }}
                >
                  <span>残念だった</span>
                  <strong>{result.disappointedCount}</strong>
                </Card>
              </div>
            </header>

            {result.items.length === 0 ? (
              <TextState variant="empty" className="avatar-review-page__empty">
                {emptyMessage}
              </TextState>
            ) : (
              <div className="avatar-review-page__list">
                {result.items.map((review) => (
                  <Card
                    key={review.id}
                    as="article"
                    variant="panel"
                    className="avatar-review-card"
                  >
                    <div className="avatar-review-card__header">
                      <div className="avatar-review-card__reviewer">
                        {review.reviewerAvatarIcon ? (
                          <img
                            src={review.reviewerAvatarIcon}
                            alt=""
                            className="avatar-review-card__reviewer-icon"
                          />
                        ) : (
                          <div
                            className="avatar-review-card__reviewer-icon avatar-review-card__reviewer-icon--fallback"
                            aria-hidden="true"
                          >
                            {review.reviewerAvatarName.trim().charAt(0) || "A"}
                          </div>
                        )}

                        <strong className="avatar-review-card__reviewer-name">
                          {review.reviewerAvatarName || "アバター"}
                        </strong>
                      </div>

                      <time className="avatar-review-card__date">
                        {formatCreatedAt(review.createdAt)}
                      </time>
                    </div>

                    <p className="avatar-review-card__comment">
                      {review.comment}
                    </p>
                  </Card>
                ))}
              </div>
            )}

            <Pagination
              page={page}
              totalPages={totalPages}
              canGoPrev={page > 1 && !loading}
              canGoNext={result.hasNext && !loading}
              onPrev={() => {
                void load(page - 1);
              }}
              onNext={() => {
                void load(page + 1);
              }}
              ariaLabel="アバター評価のページ送り"
            />
          </>
        ) : null}
      </section>
    </Layout>
  );
}