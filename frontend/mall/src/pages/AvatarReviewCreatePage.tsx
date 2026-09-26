// frontend/mall/src/pages/AvatarReviewCreatePage.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import Button from "../components/ui/Button";
import Radio from "../components/ui/Radio";
import SectionCard from "../components/ui/SectionCard";
import TextState from "../components/ui/TextState";
import {
  createAvatarReview,
  fetchAvatarReviewStatus,
  type AvatarReviewEvaluation,
  type AvatarReviewStatusResponse,
} from "../features/avatar-review/api/avatarReviewApi";
import { getPublicAvatar } from "../features/avatar/api/avatarApi";

import "../styles/page-layout.css";
import "../styles/avatar-review-page.css";

const MAX_COMMENT_LENGTH = 500;

function parseOrderItemIndex(value: string): number | null {
  const normalized = value.trim();

  if (!/^\d+$/.test(normalized)) {
    return null;
  }

  const parsed = Number(normalized);

  if (!Number.isSafeInteger(parsed) || parsed < 0) {
    return null;
  }

  return parsed;
}

function countCharacters(value: string): number {
  return Array.from(value).length;
}

export default function AvatarReviewCreatePage() {
  const navigate = useNavigate();
  const {
    orderId = "",
    itemIndex = "",
  } = useParams<{
    orderId: string;
    itemIndex: string;
  }>();

  const [status, setStatus] =
    useState<AvatarReviewStatusResponse | null>(null);
  const [revieweeAvatarName, setRevieweeAvatarName] =
    useState("");
  const [evaluation, setEvaluation] =
    useState<AvatarReviewEvaluation>("good");
  const [comment, setComment] = useState("");
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitError, setSubmitError] =
    useState<string | null>(null);
  const [submitted, setSubmitted] = useState(false);

  const normalizedOrderId = orderId.trim();
  const parsedItemIndex = useMemo(
    () => parseOrderItemIndex(itemIndex),
    [itemIndex],
  );

  const commentLength = useMemo(
    () => countCharacters(comment),
    [comment],
  );

  const canSubmit =
    status?.eligible === true &&
    status.reviewed === false &&
    !submitted &&
    !submitting &&
    comment.trim().length > 0 &&
    commentLength <= MAX_COMMENT_LENGTH;

  const load = useCallback(async () => {
    if (!normalizedOrderId || parsedItemIndex === null) {
      setError("評価対象の取引情報が正しくありません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const nextStatus = await fetchAvatarReviewStatus({
        orderId: normalizedOrderId,
        orderItemIndex: parsedItemIndex,
      });

      setStatus(nextStatus);

      if (nextStatus.reviewed) {
        setRevieweeAvatarName("");
        return;
      }

      const revieweeAvatarId =
        nextStatus.revieweeAvatarId.trim();

      if (!revieweeAvatarId) {
        setRevieweeAvatarName("");
        return;
      }

      try {
        const avatar = await getPublicAvatar({
          avatarId: revieweeAvatarId,
        });

        setRevieweeAvatarName(
          avatar?.avatarName?.trim() ?? "",
        );
      } catch {
        setRevieweeAvatarName("");
      }
    } catch (caughtError) {
      setStatus(null);
      setRevieweeAvatarName("");
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : "評価対象の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [
    normalizedOrderId,
    parsedItemIndex,
  ]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleSubmit = useCallback(async () => {
    if (
      !canSubmit ||
      parsedItemIndex === null
    ) {
      return;
    }

    setSubmitting(true);
    setSubmitError(null);

    try {
      await createAvatarReview({
        orderId: normalizedOrderId,
        orderItemIndex: parsedItemIndex,
        evaluation,
        comment,
      });

      setSubmitted(true);
      setStatus((current) => {
        if (!current) {
          return current;
        }

        return {
          ...current,
          reviewed: true,
        };
      });
    } catch (caughtError) {
      setSubmitError(
        caughtError instanceof Error
          ? caughtError.message
          : "評価の投稿に失敗しました。",
      );
    } finally {
      setSubmitting(false);
    }
  }, [
    canSubmit,
    comment,
    evaluation,
    normalizedOrderId,
    parsedItemIndex,
  ]);

  const handleBackToOrder = useCallback(() => {
    if (!normalizedOrderId) {
      navigate("/wallet");
      return;
    }

    navigate(
      `/orders/${encodeURIComponent(normalizedOrderId)}`,
    );
  }, [
    navigate,
    normalizedOrderId,
  ]);

  return (
    <Layout
      title="AMOL"
      mode="mypage"
    >
      <section className="content-page-section avatar-review-page">
        <header className="avatar-review-page__header">
          <h1 className="avatar-review-page__title">
            アバターを評価
          </h1>

          {!loading &&
          !error &&
          status &&
          !status.reviewed &&
          !submitted ? (
            <TextState>
              {revieweeAvatarName
                ? `${revieweeAvatarName}との取引を評価してください。`
                : "取引相手のアバターを評価してください。"}
            </TextState>
          ) : null}
        </header>

        {loading ? (
          <TextState
            variant="loading"
            className="avatar-review-page__message"
          >
            評価対象を確認しています...
          </TextState>
        ) : null}

        {!loading && error ? (
          <>
            <TextState
              variant="error"
              className="avatar-review-page__message"
            >
              {error}
            </TextState>

            <Button
              variant="secondary"
              onClick={() => {
                void load();
              }}
            >
              再読み込み
            </Button>
          </>
        ) : null}

        {!loading &&
        !error &&
        status &&
        (status.reviewed || submitted) ? (
          <SectionCard>
            <TextState variant="success">
              {submitted
                ? "評価を投稿しました。"
                : "この取引の評価は投稿済みです。"}
            </TextState>

            <Button
              variant="secondary"
              onClick={handleBackToOrder}
            >
              注文詳細へ戻る
            </Button>
          </SectionCard>
        ) : null}

        {!loading &&
        !error &&
        status &&
        !status.eligible ? (
          <SectionCard>
            <TextState variant="error">
              この取引はアバター評価の対象ではありません。
            </TextState>

            <Button
              variant="secondary"
              onClick={handleBackToOrder}
            >
              注文詳細へ戻る
            </Button>
          </SectionCard>
        ) : null}

        {!loading &&
        !error &&
        status?.eligible === true &&
        !status.reviewed &&
        !submitted ? (
          <SectionCard>
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void handleSubmit();
              }}
            >
              <fieldset
                disabled={submitting}
                style={{
                  margin: 0,
                  padding: 0,
                  border: 0,
                }}
              >
                <legend>
                  取引はいかがでしたか？
                </legend>

                <Radio
                  id="avatar-review-good"
                  name="avatar-review-evaluation"
                  value="good"
                  checked={evaluation === "good"}
                  label="良かった"
                  onChange={() => {
                    setEvaluation("good");
                  }}
                />

                <Radio
                  id="avatar-review-disappointed"
                  name="avatar-review-evaluation"
                  value="disappointed"
                  checked={
                    evaluation === "disappointed"
                  }
                  label="残念だった"
                  onChange={() => {
                    setEvaluation("disappointed");
                  }}
                />
              </fieldset>

              <div>
                <label htmlFor="avatar-review-comment">
                  コメント
                </label>

                <textarea
                  id="avatar-review-comment"
                  value={comment}
                  disabled={submitting}
                  rows={6}
                  placeholder="取引についてコメントを入力してください。"
                  onChange={(event) => {
                    setComment(event.target.value);
                  }}
                />

                <TextState
                  variant={
                    commentLength > MAX_COMMENT_LENGTH
                      ? "error"
                      : "muted"
                  }
                >
                  {commentLength}/{MAX_COMMENT_LENGTH}文字
                </TextState>
              </div>

              {submitError ? (
                <TextState variant="error">
                  {submitError}
                </TextState>
              ) : null}

              <Button
                type="submit"
                size="lg"
                fullWidth
                disabled={!canSubmit}
              >
                {submitting
                  ? "投稿中..."
                  : "評価を投稿"}
              </Button>
            </form>
          </SectionCard>
        ) : null}
      </section>
    </Layout>
  );
}