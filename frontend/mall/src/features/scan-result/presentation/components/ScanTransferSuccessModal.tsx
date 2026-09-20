// frontend/mall/src/features/scan-result/presentation/components/ScanTransferSuccessModal.tsx

import { useEffect, useState } from "react";

import Button from "../../../../components/ui/Button";
import Chip from "../../../../components/ui/Chip";
import Modal, {
  ModalBody,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import Textbox from "../../../../components/ui/Textbox";
import TextState from "../../../../components/ui/TextState";

type ReviewEvaluation = "good" | "disappointed";

type Props = {
  open: boolean;
  loading: boolean;
  error: string | null;
  canOpenContents: boolean;
  onClose: () => void;
  onOpenContents: () => void;
  reviewEnabled?: boolean;
  reviewSubmitting?: boolean;
  reviewSubmitted?: boolean;
  reviewError?: string | null;
  onSubmitReview?: (
    evaluation: ReviewEvaluation,
    comment: string,
  ) => void | Promise<void>;
};

const MAX_REVIEW_COMMENT_LENGTH = 500;

export default function ScanTransferSuccessModal({
  open,
  loading,
  error,
  canOpenContents,
  onClose,
  onOpenContents,
  reviewEnabled = false,
  reviewSubmitting = false,
  reviewSubmitted = false,
  reviewError = null,
  onSubmitReview,
}: Props) {
  const [reviewEvaluation, setReviewEvaluation] =
    useState<ReviewEvaluation | null>(null);
  const [reviewComment, setReviewComment] = useState("");

  useEffect(() => {
    if (!open) {
      setReviewEvaluation(null);
      setReviewComment("");
    }
  }, [open]);

  const normalizedComment = reviewComment.trim();
  const canSubmitReview =
    reviewEnabled &&
    !reviewSubmitting &&
    !reviewSubmitted &&
    reviewEvaluation !== null &&
    normalizedComment.length > 0 &&
    onSubmitReview != null;

  const handleSubmitReview = async () => {
    if (
      !canSubmitReview ||
      reviewEvaluation === null ||
      onSubmitReview == null
    ) {
      return;
    }

    await onSubmitReview(reviewEvaluation, normalizedComment);
  };

  return (
    <Modal
      open={open}
      onClose={onClose}
      size="md"
      mobilePosition="bottom"
      ariaLabelledBy="scan-transfer-success-modal-title"
      ariaBusy={loading || reviewSubmitting}
    >
      <ModalHeader onClose={onClose}>
        <ModalTitle id="scan-transfer-success-modal-title">
          トークン移譲完了
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        {loading ? (
          <TextState variant="loading">
            移譲処理中です...
          </TextState>
        ) : error ? (
          <TextState variant="error">
            {error}
          </TextState>
        ) : (
          <>
            <TextState>
              トークンの移譲が完了しました。
            </TextState>

            {reviewEnabled ? (
              <section
                className="scan-transfer-modal__review"
                aria-label="取引相手への評価"
              >
                <h3 className="scan-transfer-modal__review-title">
                  取引相手への評価
                </h3>

                {reviewSubmitted ? (
                  <TextState variant="success">
                    評価を投稿しました。ありがとうございます。
                  </TextState>
                ) : (
                  <>
                    <p className="scan-transfer-modal__review-description">
                      商品を受け取るまでの対応はいかがでしたか？
                    </p>

                    <div
                      className="scan-transfer-modal__review-options"
                      role="group"
                      aria-label="評価"
                    >
                      <Chip
                        selected={reviewEvaluation === "good"}
                        disabled={reviewSubmitting}
                        onClick={() => {
                          setReviewEvaluation("good");
                        }}
                      >
                        良かった
                      </Chip>

                      <Chip
                        selected={reviewEvaluation === "disappointed"}
                        disabled={reviewSubmitting}
                        onClick={() => {
                          setReviewEvaluation("disappointed");
                        }}
                      >
                        残念だった
                      </Chip>
                    </div>

                    <Textbox
                      label="コメント"
                      value={reviewComment}
                      maxLength={MAX_REVIEW_COMMENT_LENGTH}
                      disabled={reviewSubmitting}
                      placeholder="取引相手の対応についてコメントしてください"
                      counterText={`${reviewComment.length}/${MAX_REVIEW_COMMENT_LENGTH}`}
                      error={reviewError ?? undefined}
                      onChange={(event) => {
                        setReviewComment(event.target.value);
                      }}
                    />

                    <Button
                      type="button"
                      fullWidth
                      disabled={!canSubmitReview}
                      onClick={() => {
                        void handleSubmitReview();
                      }}
                    >
                      {reviewSubmitting ? "投稿中..." : "評価を投稿"}
                    </Button>
                  </>
                )}
              </section>
            ) : null}
          </>
        )}
      </ModalBody>

      <ModalFooter className="scan-transfer-modal__footer">
        {canOpenContents && !loading && !error ? (
          <Button
            type="button"
            fullWidth
            onClick={onOpenContents}
          >
            コンテンツを見る
          </Button>
        ) : null}

        <Button
          type="button"
          variant="secondary"
          fullWidth
          onClick={onClose}
        >
          閉じる
        </Button>
      </ModalFooter>
    </Modal>
  );
}