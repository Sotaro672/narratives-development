// frontend/mall/src/features/identityVerification/hooks/useIdentityVerification.ts

import {
  useCallback,
  useEffect,
  useState,
} from "react";

import {
  fetchIdentityVerification,
  verifyIdentityWithMockMyNumberCard,
} from "../api/identityVerificationApi";
import {
  isIdentityVerified,
  type IdentityVerification,
} from "../types";

type UseIdentityVerificationOptions = {
  enabled?: boolean;
};

function getIdentityVerificationErrorMessage(
  error: unknown,
  fallback: string,
): string {
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }

  return fallback;
}

export function useIdentityVerification(
  options: UseIdentityVerificationOptions = {},
) {
  const { enabled = true } = options;

  const [verification, setVerification] =
    useState<IdentityVerification | null>(null);
  const [isLoading, setIsLoading] = useState(enabled);
  const [isVerifying, setIsVerifying] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const load = useCallback(
    async (
      signal?: AbortSignal,
    ): Promise<IdentityVerification | null> => {
      if (!enabled) {
        setVerification(null);
        setIsLoading(false);
        setErrorMessage("");
        return null;
      }

      setIsLoading(true);
      setErrorMessage("");

      try {
        const result = await fetchIdentityVerification({
          signal,
        });

        if (signal?.aborted) {
          return null;
        }

        setVerification(result);
        return result;
      } catch (error) {
        if (signal?.aborted) {
          return null;
        }

        setErrorMessage(
          getIdentityVerificationErrorMessage(
            error,
            "本人確認情報を取得できませんでした。",
          ),
        );

        return null;
      } finally {
        if (!signal?.aborted) {
          setIsLoading(false);
        }
      }
    },
    [enabled],
  );

  useEffect(() => {
    if (!enabled) {
      setVerification(null);
      setIsLoading(false);
      setErrorMessage("");
      return;
    }

    const controller = new AbortController();

    void load(controller.signal);

    return () => {
      controller.abort();
    };
  }, [enabled, load]);

  const reload = useCallback(
    async (): Promise<IdentityVerification | null> => {
      return load();
    },
    [load],
  );

  const verify = useCallback(
    async (): Promise<IdentityVerification | null> => {
      if (!enabled || isVerifying) {
        return null;
      }

      if (isIdentityVerified(verification)) {
        return verification;
      }

      setIsVerifying(true);
      setErrorMessage("");

      try {
        const result =
          await verifyIdentityWithMockMyNumberCard();

        setVerification(result);
        return result;
      } catch (error) {
        setErrorMessage(
          getIdentityVerificationErrorMessage(
            error,
            "本人確認を完了できませんでした。",
          ),
        );

        return null;
      } finally {
        setIsVerifying(false);
      }
    },
    [
      enabled,
      isVerifying,
      verification,
    ],
  );

  const clearError = useCallback(() => {
    setErrorMessage("");
  }, []);

  return {
    verification,
    isVerified: isIdentityVerified(verification),
    isLoading,
    isVerifying,
    errorMessage,
    load,
    reload,
    verify,
    clearError,
  };
}

export default useIdentityVerification;