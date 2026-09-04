// frontend/amol/src/features/token-commnet/components/TokenCommentReplyForm.tsx

import type { ChangeEvent } from "react";

type TokenCommentReplyFormProps = {
  value: string;
  replyPosting: boolean;
  placeholder?: string;
  submitLabel?: string;
  postingLabel?: string;
  cancelLabel?: string;
  onChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: () => void | Promise<void>;
};

export default function TokenCommentReplyForm({
  value,
  replyPosting,
  placeholder = "返信を書く…",
  submitLabel = "返信を投稿",
  postingLabel = "投稿中...",
  cancelLabel = "キャンセル",
  onChange,
  onCancel,
  onSubmit,
}: TokenCommentReplyFormProps) {
  const trimmedValue = value.trim();
  const canSubmit = !replyPosting && trimmedValue.length > 0;

  const handleChange = (event: ChangeEvent<HTMLTextAreaElement>) => {
    onChange(event.target.value);
  };

  const handleSubmit = () => {
    if (!canSubmit) {
      return;
    }

    void onSubmit();
  };

  return (
    <div className="token-comment-reply-form">
      <textarea
        className="token-comment-reply-form__textarea"
        value={value}
        rows={3}
        disabled={replyPosting}
        placeholder={placeholder}
        onChange={handleChange}
      />

      <div className="token-comment-reply-form__actions">
        <button
          type="button"
          className="token-comment-reply-form__button token-comment-reply-form__button--secondary"
          disabled={replyPosting}
          onClick={onCancel}
        >
          {cancelLabel}
        </button>

        <button
          type="button"
          className="token-comment-reply-form__button"
          disabled={!canSubmit}
          onClick={handleSubmit}
        >
          {replyPosting ? postingLabel : submitLabel}
        </button>
      </div>
    </div>
  );
}