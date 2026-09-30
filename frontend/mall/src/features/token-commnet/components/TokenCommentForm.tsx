// frontend/mall/src/features/token-commnet/components/TokenCommentForm.tsx

import type {
  ChangeEvent,
  KeyboardEvent,
} from "react";

import Textbox from "../../../components/ui/Textbox";

type TokenCommentFormProps = {
  value: string;
  posting: boolean;
  loading?: boolean;
  rows?: number;
  placeholder?: string;
  onChange: (value: string) => void;
  onSubmit: () => void | Promise<void>;
};

export default function TokenCommentForm({
  value,
  posting,
  loading = false,
  rows = 4,
  placeholder = "コメントを書く…",
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

  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (
      event.key !== "Enter" ||
      event.shiftKey ||
      event.nativeEvent.isComposing
    ) {
      return;
    }

    event.preventDefault();
    handleSubmit();
  };

  return (
    <div className="token-comment-form">
      <Textbox
        className="token-comment-form__textarea"
        value={value}
        rows={rows}
        disabled={disabled}
        placeholder={placeholder}
        onChange={handleChange}
        onKeyDown={handleKeyDown}
      />
    </div>
  );
}