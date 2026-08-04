// frontend/amol/src/features/market/presentation/components/MarketReviewSection.tsx

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import type {
  ProductBlueprintReviewPage,
} from "../../../shared/types/review";

import MarketReviewAvatar from "./MarketReviewAvatar";

type MarketReviewSectionProps = {
  reviews:
    ProductBlueprintReviewPage | null;
  loading: boolean;
  error: string;
};

function formatReviewDate(
  value:
    string | undefined,
): string {
  const text =
    textOrEmpty(value);

  if (!text) {
    return "";
  }

  const date =
    new Date(text);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return "";
  }

  return new Intl.DateTimeFormat(
    "ja-JP",
    {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
    },
  ).format(date);
}

function getRatingStars(
  value:
    number | undefined,
): string {
  const rating =
    Math.max(
      0,
      Math.min(
        5,
        Math.trunc(
          Number(
            value ?? 0,
          ),
        ),
      ),
    );

  if (rating <= 0) {
    return "評価なし";
  }

  return (
    "★".repeat(rating) +
    "☆".repeat(
      5 - rating,
    )
  );
}

export default function MarketReviewSection({
  reviews,
  loading,
  error,
}: MarketReviewSectionProps) {
  return (
    <section className="market-detail-page__reviews">
      <div className="market-detail-page__reviews-header">
        <h2>
          レビュー
        </h2>

        {loading ? (
          <span className="market-detail-page__reviews-status">
            読み込み中...
          </span>
        ) : null}
      </div>

      {error ? (
        <p
          className="market-detail-page__reviews-error"
          role="alert"
        >
          {error}
        </p>
      ) : null}

      {!loading &&
      !error &&
      reviews?.items.length ? (
        <div className="market-detail-page__review-list">
          {reviews.items.map(
            (review) => {
              const reviewTitle =
                textOrEmpty(
                  review.title,
                );

              const body =
                textOrEmpty(
                  review.body,
                );

              const reviewedAt =
                formatReviewDate(
                  review.reviewedAt,
                );

              return (
                <article
                  className="market-detail-page__review"
                  key={review.id}
                >
                  <div className="market-detail-page__review-top">
                    <MarketReviewAvatar
                      review={
                        review
                      }
                    />

                    <span className="market-detail-page__review-rating">
                      {getRatingStars(
                        review.rating,
                      )}
                    </span>
                  </div>

                  {reviewTitle ? (
                    <h3 className="market-detail-page__review-title">
                      {reviewTitle}
                    </h3>
                  ) : null}

                  {body ? (
                    <p className="market-detail-page__review-body">
                      {body}
                    </p>
                  ) : null}

                  {reviewedAt ? (
                    <time
                      className="market-detail-page__review-date"
                      dateTime={
                        review.reviewedAt
                      }
                    >
                      {reviewedAt}
                    </time>
                  ) : null}
                </article>
              );
            },
          )}
        </div>
      ) : null}

      {!loading &&
      !error &&
      (!reviews ||
        reviews.items.length ===
          0) ? (
        <p className="market-detail-page__reviews-empty">
          まだレビューはありません。
        </p>
      ) : null}
    </section>
  );
}