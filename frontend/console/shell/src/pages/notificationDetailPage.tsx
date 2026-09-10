// frontend/console/shell/src/pages/notificationDetailPage.tsx

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  getNewsApi,
  markNewsReadApi,
} from "../features/notification/infrastructure/newsApi";
import {
  getReportDecisionNotificationApi,
  markReportDecisionNotificationReadApi,
} from "../features/notification/infrastructure/reportDecisionNotificationApi";
import {
  emitNewsNotificationChanged,
  emitReportDecisionNotificationChanged,
} from "../features/notification/presentation/notificationEvent";
import {
  toReportDecisionNotificationViewModel,
  type ReportDecisionNotificationViewModel,
} from "../features/notification/presentation/model/reportDecisionNotification";
import PageStyle from "../layout/PageStyle/PageStyle";
import type { News } from "../shared/types/news";

import "../styles/notificationDetail.css";

type NotificationDetailKind = "news" | "reportDecision";

type NotificationDetailPageProps = {
  kind: NotificationDetailKind;
};

type NotificationDetail =
  | {
      kind: "news";
      news: News;
    }
  | {
      kind: "reportDecision";
      notification: ReportDecisionNotificationViewModel;
    };

function resolveErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }

  return "通知詳細の取得に失敗しました。";
}

export default function NotificationDetailPage({
  kind,
}: NotificationDetailPageProps) {
  const navigate = useNavigate();
  const { notificationId = "" } = useParams<{
    notificationId?: string;
  }>();

  const [detail, setDetail] = useState<NotificationDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const processedKeyRef = useRef<string | null>(null);

  const normalizedNotificationId = useMemo(
    () => notificationId.trim(),
    [notificationId],
  );

  const load = useCallback(async () => {
    if (!normalizedNotificationId) {
      setDetail(null);
      setError("通知IDを取得できませんでした。");
      return;
    }

    const processedKey = `${kind}:${normalizedNotificationId}`;

    if (processedKeyRef.current === processedKey) {
      return;
    }

    processedKeyRef.current = processedKey;
    setLoading(true);
    setError(null);

    try {
      if (kind === "news") {
        let news = await getNewsApi(normalizedNotificationId);

        if (!news.isRead) {
          const read = await markNewsReadApi(normalizedNotificationId);

          news = {
            ...news,
            isRead: true,
            readAt: read.readAt,
          };

          emitNewsNotificationChanged();
        }

        setDetail({
          kind: "news",
          news,
        });

        return;
      }

      let notification = await getReportDecisionNotificationApi(
        normalizedNotificationId,
      );

      if (!notification.isRead) {
        notification = await markReportDecisionNotificationReadApi(
          normalizedNotificationId,
        );

        emitReportDecisionNotificationChanged();
      }

      setDetail({
        kind: "reportDecision",
        notification: toReportDecisionNotificationViewModel(notification),
      });
    } catch (loadError) {
      processedKeyRef.current = null;
      setDetail(null);
      setError(resolveErrorMessage(loadError));
    } finally {
      setLoading(false);
    }
  }, [kind, normalizedNotificationId]);

  useEffect(() => {
    void load();
  }, [load]);

  const handleBack = useCallback(() => {
    navigate("/notifications");
  }, [navigate]);

  if (loading && !detail) {
    return (
      <PageStyle layout="single" title="通知詳細" onBack={handleBack}>
        <div className="notification-detail">
          <p className="notification-detail__message">読み込み中です。</p>
        </div>
      </PageStyle>
    );
  }

  if (error) {
    return (
      <PageStyle layout="single" title="通知詳細" onBack={handleBack}>
        <div className="notification-detail">
          <p
            className="notification-detail__message notification-detail__message--error"
            role="alert"
          >
            {error}
          </p>
        </div>
      </PageStyle>
    );
  }

  if (!detail) {
    return (
      <PageStyle layout="single" title="通知詳細" onBack={handleBack}>
        <div className="notification-detail">
          <p className="notification-detail__message">
            表示可能な通知がありません。
          </p>
        </div>
      </PageStyle>
    );
  }

  if (detail.kind === "news") {
    const { news } = detail;

    return (
      <PageStyle layout="single" title={news.title} onBack={handleBack}>
        <article className="notification-detail">
          <div className="notification-detail__body">{news.body}</div>

          {news.image ? (
            <div className="notification-detail__image-wrap">
              <img
                className="notification-detail__image"
                src={news.image.fileUrl}
                alt={news.image.alt || news.title}
              />
            </div>
          ) : null}
        </article>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="single"
      title={detail.notification.title}
      onBack={handleBack}
    >
      <article className="notification-detail">
        <div className="notification-detail__body">
          {detail.notification.body}
        </div>
      </article>
    </PageStyle>
  );
}