// frontend/mall/src/features/token-commnet/components/TokenCommentReplyForm.tsx

import type { ChangeEvent } from "react";

import Button from "../../../components/ui/Button";
import Textbox from "../../../components/ui/Textbox";

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
      <Textbox value={value} rows={3} disabled={replyPosting} placeholder={placeholder} onChange={handleChange} />

      <div className="token-comment-reply-form__actions">
        <Button type="button" variant="secondary" size="sm" disabled={replyPosting} onClick={onCancel}>
          {cancelLabel}
        </Button>

        <Button type="button" size="sm" disabled={!canSubmit} onClick={handleSubmit}>
          {replyPosting ? postingLabel : submitLabel}
        </Button>
      </div>
    </div>
  );
}