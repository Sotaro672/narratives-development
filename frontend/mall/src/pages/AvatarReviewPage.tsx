// frontend/mall/src/pages/AvatarReviewPage.tsx

import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import Card from "../components/ui/Card";
import Pagination from "../components/ui/Pagination";
import TextState from "../components/ui/TextState";
import {
  fetchAvatarReviews,
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
  const navigate = useNavigate();
  const { avatarId = "" } = useParams<{ avatarId: string }>();

  const [avatarName, setAvatarName] = useState("");
  const [result, setResult] = useState<AvatarReviewPageResponse | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(
    async (nextPage: number) => {
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
    [avatarId],
  );

  useEffect(() => {
    void load(1);
  }, [load]);

  const totalPages = result
    ? Math.max(1, Math.ceil(result.total / result.perPage))
    : 1;

  return (
    <Layout
      title={avatarName ? `${avatarName}の評価` : "評価"}
      mode="mypage"
      showBackButton
      onBackButtonClick={() => navigate(-1)}
    >
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
                <Card variant="panel" className="avatar-review-page__summary-item">
                  <span>良かった</span>
                  <strong>{result.goodCount}</strong>
                </Card>

                <Card variant="panel" className="avatar-review-page__summary-item">
                  <span>残念だった</span>
                  <strong>{result.disappointedCount}</strong>
                </Card>
              </div>
            </header>

            {result.items.length === 0 ? (
              <TextState variant="empty" className="avatar-review-page__empty">
                まだ評価はありません。
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
                      <strong
                        className={
                          review.evaluation === "good"
                            ? "avatar-review-card__evaluation avatar-review-card__evaluation--good"
                            : "avatar-review-card__evaluation avatar-review-card__evaluation--disappointed"
                        }
                      >
                        {review.evaluation === "good"
                          ? "良かった"
                          : "残念だった"}
                      </strong>

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