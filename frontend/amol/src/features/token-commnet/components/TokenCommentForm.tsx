// frontend/amol/src/features/token-commnet/components/TokenCommentForm.tsx

import type { ChangeEvent } from "react";

type TokenCommentFormProps = {
  value: string;
  posting: boolean;
  loading?: boolean;
  placeholder?: string;
  buttonLabel?: string;
  postingLabel?: string;
  onChange: (value: string) => void;
  onSubmit: () => void | Promise<void>;
};

export default function TokenCommentForm({
  value,
  posting,
  loading = false,
  placeholder = "コメントを書く…",
  buttonLabel = "投稿",
  postingLabel = "投稿中...",
  onChange,
  onSubmit,
}: TokenCommentFormProps) {
  const trimmedValue = value.trim();
  const disabled = posting || loading;
  const canSubmit = !disabled && trimmedValue.length > 0;

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
    <div className="token-comment-form">
      <textarea
        className="token-comment-form__textarea"
        value={value}
        rows={4}
        disabled={disabled}
        placeholder={placeholder}
        onChange={handleChange}
      />

      <button
        type="button"
        className="token-comment-form__button"
        disabled={!canSubmit}
        onClick={handleSubmit}
      >
        {posting ? postingLabel : buttonLabel}
      </button>
    </div>
  );
}