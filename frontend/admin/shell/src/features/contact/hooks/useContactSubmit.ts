// frontend/admin/shell/src/features/contact/hooks/useContactSubmit.ts
import { useCallback, useRef, useState } from "react";

type ContactSubmitOperation<TResult> = () => Promise<TResult>;

type UseContactSubmitOptions<TResult> = {
  onSuccess?: (result: TResult) => void | Promise<void>;
  onError?: (error: Error) => void | Promise<void>;
};

export function useContactSubmit<TResult = void>(
  options: UseContactSubmitOptions<TResult> = {},
) {
  const { onSuccess, onError } = options;
  const submittingRef = useRef(false);
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const submit = useCallback(
    async (operation: ContactSubmitOperation<TResult>): Promise<TResult | null> => {
      if (submittingRef.current) {
        return null;
      }

      submittingRef.current = true;
      setSubmitting(true);
      setSubmitError(null);

      try {
        const result = await operation();

        if (onSuccess) {
          await onSuccess(result);
        }

        return result;
      } catch (cause) {
        const error =
          cause instanceof Error
            ? cause
            : new Error("問い合わせ処理に失敗しました。");

        setSubmitError(error.message);

        if (onError) {
          await onError(error);
        }

        return null;
      } finally {
        submittingRef.current = false;
        setSubmitting(false);
      }
    },
    [onSuccess, onError],
  );

  const clearSubmitError = useCallback(() => {
    setSubmitError(null);
  }, []);

  return {
    submitting,
    submitError,
    submit,
    clearSubmitError,
  };
}