// frontend/admin/shell/src/features/contact/hooks/useContactSubmit.ts
import {
  useCallback,
  useState,
} from "react";

type ContactSubmitOperation<TResult> =
  () => Promise<TResult>;

type UseContactSubmitOptions<TResult> = {
  onSuccess?: (result: TResult) => void | Promise<void>;
  onError?: (error: Error) => void | Promise<void>;
};

export function useContactSubmit<TResult = void>(
  options: UseContactSubmitOptions<TResult> = {},
) {
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const submit = useCallback(
    async (
      operation: ContactSubmitOperation<TResult>,
    ): Promise<TResult | null> => {
      if (submitting) {
        return null;
      }

      setSubmitting(true);
      setSubmitError(null);

      try {
        const result = await operation();

        if (options.onSuccess) {
          await options.onSuccess(result);
        }

        return result;
      } catch (cause) {
        const error =
          cause instanceof Error
            ? cause
            : new Error(
                "問い合わせ処理に失敗しました。",
              );

        setSubmitError(error.message);

        if (options.onError) {
          await options.onError(error);
        }

        return null;
      } finally {
        setSubmitting(false);
      }
    },
    [
      submitting,
      options.onSuccess,
      options.onError,
    ],
  );

  const clearSubmitError =
    useCallback(() => {
      setSubmitError(null);
    }, []);

  return {
    submitting,
    submitError,
    submit,
    clearSubmitError,
  };
}