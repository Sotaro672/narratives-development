// frontend/console/shell/src/features/inquiry/presentation/hooks/useInquiryDetailPage.ts

import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import { useAuth } from "../../../../auth/presentation/hook/useCurrentMember";

import {
  getInquiryHTTP,
  reopenInquiryHTTP,
  resolveInquiryHTTP,
} from "../../infrastructure/inquiryRepositoryHTTP";

import {
  isClosedStatus,
  isResolvedStatus,
} from "../utils/inquiryStatus";

import type {
  InquiryDetail as InquiryDetailDTO,
} from "../../../../shared/types/inquiry";

export const INQUIRY_READ_STATE_CHANGED_EVENT =
  "inquiry:read-state-changed";

type LoadInquiryDetailOptions = {
  showLoading: boolean;
  clearDetailOnError: boolean;
};

export type UseInquiryDetailPageResult = {
  inquiryId: string;
  memberId: string;

  detail: InquiryDetailDTO | null;
  loading: boolean;
  statusUpdating: boolean;
  errorMessage: string | null;

  onBack: () => void;
  reloadDetail: () => Promise<InquiryDetailDTO | null>;
  clearErrorMessage: () => void;
  onToggleStatus: () => Promise<void>;
};

function normalizeID(
  value: string | null | undefined,
): string {
  return String(value ?? "").trim();
}

function replaceDetailInquiry(
  detail: InquiryDetailDTO,
  inquiry: InquiryDetailDTO["inquiry"],
): InquiryDetailDTO {
  return {
    ...detail,
    inquiry,
  };
}

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  return error instanceof Error
    ? error.message
    : fallbackMessage;
}

export function useInquiryDetailPage(): UseInquiryDetailPageResult {
  const navigate = useNavigate();

  const {
    inquiryId: inquiryIdParam,
  } = useParams<{
    inquiryId?: string;
  }>();

  const {
    currentMember,
  } = useAuth();

  const inquiryId = normalizeID(
    inquiryIdParam,
  );

  const memberId = normalizeID(
    currentMember?.id,
  );

  const [
    detail,
    setDetail,
  ] = useState<InquiryDetailDTO | null>(
    null,
  );

  const [
    loading,
    setLoading,
  ] = useState(true);

  const [
    statusUpdating,
    setStatusUpdating,
  ] = useState(false);

  const [
    errorMessage,
    setErrorMessage,
  ] = useState<string | null>(
    null,
  );

  const mountedRef = useRef(false);
  const latestRequestIdRef = useRef(0);

  const onBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const clearErrorMessage =
    useCallback(() => {
      setErrorMessage(null);
    }, []);

  const loadDetail = useCallback(
    async (
      options: LoadInquiryDetailOptions,
    ): Promise<InquiryDetailDTO | null> => {
      const requestId =
        latestRequestIdRef.current + 1;

      latestRequestIdRef.current =
        requestId;

      if (!inquiryId) {
        if (mountedRef.current) {
          setDetail(null);

          setErrorMessage(
            "問い合わせIDが指定されていません。",
          );

          setLoading(false);
        }

        return null;
      }

      if (options.showLoading) {
        setLoading(true);
      }

      setErrorMessage(null);

      try {
        const result =
          await getInquiryHTTP(
            inquiryId,
          );

        if (
          !mountedRef.current ||
          requestId !==
            latestRequestIdRef.current
        ) {
          return null;
        }

        setDetail(result);

        window.dispatchEvent(
          new Event(
            INQUIRY_READ_STATE_CHANGED_EVENT,
          ),
        );

        return result;
      } catch (error) {
        if (
          !mountedRef.current ||
          requestId !==
            latestRequestIdRef.current
        ) {
          return null;
        }

        setErrorMessage(
          getErrorMessage(
            error,
            "問い合わせ詳細の取得に失敗しました",
          ),
        );

        if (
          options.clearDetailOnError
        ) {
          setDetail(null);
        }

        return null;
      } finally {
        if (
          options.showLoading &&
          mountedRef.current &&
          requestId ===
            latestRequestIdRef.current
        ) {
          setLoading(false);
        }
      }
    },
    [inquiryId],
  );

  useEffect(() => {
    mountedRef.current = true;

    void loadDetail({
      showLoading: true,
      clearDetailOnError: true,
    });

    return () => {
      mountedRef.current = false;

      latestRequestIdRef.current += 1;
    };
  }, [loadDetail]);

  const reloadDetail =
    useCallback(
      async (): Promise<InquiryDetailDTO | null> => {
        return loadDetail({
          showLoading: false,
          clearDetailOnError: false,
        });
      },
      [loadDetail],
    );

  const onToggleStatus =
    useCallback(
      async (): Promise<void> => {
        if (
          statusUpdating ||
          !detail ||
          !inquiryId
        ) {
          return;
        }

        if (
          isClosedStatus(
            detail.inquiry.status,
          )
        ) {
          return;
        }

        if (!memberId) {
          setErrorMessage(
            "メンバーIDが取得できません。ログインし直してください。",
          );

          return;
        }

        setStatusUpdating(true);
        setErrorMessage(null);

        try {
          const updatedInquiry =
            isResolvedStatus(
              detail.inquiry.status,
            )
              ? await reopenInquiryHTTP(
                  inquiryId,
                  {
                    memberId,
                  },
                )
              : await resolveInquiryHTTP(
                  inquiryId,
                  {
                    memberId,
                  },
                );

          if (!mountedRef.current) {
            return;
          }

          setDetail((current) => {
            if (!current) {
              return current;
            }

            return replaceDetailInquiry(
              current,
              updatedInquiry,
            );
          });
        } catch (error) {
          if (!mountedRef.current) {
            return;
          }

          setErrorMessage(
            getErrorMessage(
              error,
              "問い合わせステータスの更新に失敗しました",
            ),
          );
        } finally {
          if (mountedRef.current) {
            setStatusUpdating(false);
          }
        }
      },
      [
        detail,
        inquiryId,
        memberId,
        statusUpdating,
      ],
    );

  return {
    inquiryId,
    memberId,

    detail,
    loading,
    statusUpdating,
    errorMessage,

    onBack,
    reloadDetail,
    clearErrorMessage,
    onToggleStatus,
  };
}