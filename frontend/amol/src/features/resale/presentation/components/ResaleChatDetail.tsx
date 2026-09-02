// frontend/amol/src/features/resale/presentation/components/ResaleChatDetail.tsx

import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import Layout from "../../../../components/layout/Layout";

import { getMyAvatar } from "../../../avatar/api/avatarApi";

import { fetchMarketResaleById } from "../../../market/infrastructure/marketResaleApi";
import { fetchMarketResaleConditionImages } from "../../../market/infrastructure/marketResaleImageApi";
import {
  createMarketResaleComment,
  deleteMarketResaleComment,
  fetchMarketResaleComments,
} from "../../../market/infrastructure/marketResaleReviewApi";

import ChatComposerModal from "../../../shared/presentation/components/ChatComposerModal";
import ChatImageGrid from "../../../shared/presentation/components/ChatImageGrid";
import ChatMessageBubble from "../../../shared/presentation/components/ChatMessageBubble";
import ChatMessageHeader from "../../../shared/presentation/components/ChatMessageHeader";
import ChatMetaSection, { type ChatMetaItem } from "../../../shared/presentation/components/ChatMetaSection";
import ChatThreadCard from "../../../shared/presentation/components/ChatThreadCard";
import { createProductModelDisplay } from "../../../shared/presentation/utils/productModelDisplay";
import type { MarketResaleListing } from "../../../shared/types/marketResale";
import type {
  ResaleConditionImage,
  ResaleListing,
  ResaleStatus,
} from "../../../shared/types/resale";
import type {
  ResaleReviewComment,
  ResaleReviewCommentPage,
} from "../../../shared/types/resaleReview";

import {
  createMyResaleComment,
  deleteMyResaleComment,
  fetchMyResaleComments,
  getMyResaleListing,
  listMyResaleConditionImages,
} from "../../api/resaleApi";
import { markMyResaleCommentsAsRead } from "../../api/resaleReviewApi";
import { updateResaleChatBadgeCount } from "../resaleChatBadgeEvents";

const COMMENT_PER_PAGE = 100;

type ResaleChatSource = "market" | "owner";

type ResaleChatRouteState = {
  source?: ResaleChatSource;
};

type ResaleChatItem = MarketResaleListing | ResaleListing;

type ResaleChatData = {
  source: ResaleChatSource;
  item: ResaleChatItem;
  images: ResaleConditionImage[];
  comments: ResaleReviewComment[];
};

type ResaleChatDetailProps = {
  resaleId: string;
  onBack: () => void;
};

function getErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  if (error instanceof Error && error.message !== "") {
    return error.message;
  }

  return fallbackMessage;
}

function getResaleStatusLabel(status: ResaleStatus): string {
  switch (status) {
    case "listing":
      return "出品中";

    case "suspended":
      return "公開停止";

    case "sold":
      return "売却済み";
  }
}

function getProductMetaItems(item: ResaleChatItem): ChatMetaItem[] {
  const model = createProductModelDisplay(item);
  const items: ChatMetaItem[] = [
    {
      label: "販売価格",
      value: `${item.price.toLocaleString("ja-JP")}円`,
    },
    {
      label: "商品の状態",
      value: item.condition,
    },
  ];

  if (model.modelNumber) {
    items.push({
      label: "モデル番号",
      value: model.modelNumber,
    });
  }

  if (model.kindLabel && model.kindLabel !== "アパレル") {
    items.push({
      label: "種別",
      value: model.kindLabel,
    });
  }

  if (model.size) {
    items.push({
      label: "サイズ",
      value: model.size,
    });
  }

  if (model.colorLabel || model.colorCssValue) {
    items.push({
      label: "カラー",
      value: model.colorLabel || model.colorCssValue,
    });
  }

  if (model.measurementsLabel !== "-") {
    items.push({
      label: "採寸",
      value: model.measurementsLabel,
    });
  }

  if (model.volumeLabel !== "-") {
    items.push({
      label: "容量",
      value: model.volumeLabel,
    });
  }

  return items;
}

function sortComments(
  comments: ResaleReviewComment[],
): ResaleReviewComment[] {
  return [...comments].sort(
    (firstComment, secondComment) =>
      new Date(firstComment.createdAt).getTime() -
      new Date(secondComment.createdAt).getTime(),
  );
}

async function fetchAllComments(
  source: ResaleChatSource,
  resaleId: string,
): Promise<ResaleReviewComment[]> {
  const comments: ResaleReviewComment[] = [];
  let page = 1;

  while (true) {
    const result: ResaleReviewCommentPage =
      source === "owner"
        ? await fetchMyResaleComments({
            resaleId,
            page,
            perPage: COMMENT_PER_PAGE,
          })
        : await fetchMarketResaleComments({
            resaleId,
            page,
            perPage: COMMENT_PER_PAGE,
          });

    comments.push(...result.items);

    if (page >= result.totalPages) {
      break;
    }

    page += 1;
  }

  return sortComments(comments);
}

async function loadOwnerChat(
  resaleId: string,
): Promise<ResaleChatData> {
  const [item, images, comments] = await Promise.all([
    getMyResaleListing(resaleId),
    listMyResaleConditionImages(resaleId),
    fetchAllComments("owner", resaleId),
  ]);

  return {
    source: "owner",
    item,
    images,
    comments,
  };
}

async function loadMarketChat(
  resaleId: string,
): Promise<ResaleChatData> {
  const [item, images, comments] = await Promise.all([
    fetchMarketResaleById(resaleId),
    fetchMarketResaleConditionImages(resaleId),
    fetchAllComments("market", resaleId),
  ]);

  return {
    source: "market",
    item,
    images,
    comments,
  };
}

async function loadResaleChat(
  resaleId: string,
  preferredSource?: ResaleChatSource,
): Promise<ResaleChatData> {
  if (preferredSource === "owner") {
    return loadOwnerChat(resaleId);
  }

  if (preferredSource === "market") {
    return loadMarketChat(resaleId);
  }

  try {
    return await loadOwnerChat(resaleId);
  } catch {
    return loadMarketChat(resaleId);
  }
}

export default function ResaleChatDetail({
  resaleId,
  onBack,
}: ResaleChatDetailProps) {
  const location = useLocation();
  const navigate = useNavigate();

  const routeState =
    location.state as ResaleChatRouteState | null;

  const preferredSource = routeState?.source;
  const normalizedResaleId = resaleId.trim();

  const [source, setSource] = useState<ResaleChatSource | null>(null);
  const [item, setItem] = useState<ResaleChatItem | null>(null);
  const [images, setImages] = useState<ResaleConditionImage[]>([]);
  const [comments, setComments] = useState<ResaleReviewComment[]>([]);
  const [viewerAvatarId, setViewerAvatarId] = useState("");

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const [isReplyModalOpen, setIsReplyModalOpen] = useState(false);
  const [replyContent, setReplyContent] = useState("");
  const [replyError, setReplyError] = useState("");
  const [postingReply, setPostingReply] = useState(false);
  const [deletingCommentId, setDeletingCommentId] = useState("");

  const loadThread = useCallback(async (): Promise<void> => {
    if (!normalizedResaleId) {
      setSource(null);
      setItem(null);
      setImages([]);
      setComments([]);
      setViewerAvatarId("");
      setError("出品IDが見つかりません。");
      setLoading(false);
      return;
    }

    setLoading(true);
    setError("");
    setReplyError("");

    try {
      const [chatData, myAvatar] = await Promise.all([
        loadResaleChat(
          normalizedResaleId,
          preferredSource,
        ),
        getMyAvatar().catch(() => null),
      ]);

      let readError = "";

      if (chatData.source === "owner") {
        try {
          const result = await markMyResaleCommentsAsRead({
            resaleId: normalizedResaleId,
          });

          if (result.markedCount > 0) {
            updateResaleChatBadgeCount(-result.markedCount);
          }
        } catch (caught) {
          readError = getErrorMessage(
            caught,
            "コメントの既読状態の更新に失敗しました。",
          );
        }
      }

      setSource(chatData.source);
      setItem(chatData.item);
      setImages(chatData.images);
      setComments(chatData.comments);
      setViewerAvatarId(
        myAvatar?.avatarId ||
          (
            chatData.source === "owner"
              ? chatData.item.avatarId
              : ""
          ),
      );

      if (readError) {
        setError(readError);
      }
    } catch (caught) {
      setSource(null);
      setItem(null);
      setImages([]);
      setComments([]);
      setViewerAvatarId("");
      setError(
        getErrorMessage(
          caught,
          "コメント内容の取得に失敗しました。",
        ),
      );
    } finally {
      setLoading(false);
    }
  }, [
    normalizedResaleId,
    preferredSource,
  ]);

  useEffect(() => {
    void loadThread();
  }, [loadThread]);

  const sortedComments = useMemo(
    () => sortComments(comments),
    [comments],
  );

  const productMetaItems = useMemo(
    () =>
      item
        ? getProductMetaItems(item)
        : [],
    [item],
  );

  const handleOpenMarketDetail = useCallback(() => {
    if (!normalizedResaleId) return;

    navigate(
      `/market/${encodeURIComponent(normalizedResaleId)}`,
    );
  }, [navigate, normalizedResaleId]);

  const title =
    item?.productName ||
    item?.tokenName ||
    "コメント";

  const canSubmitReply =
    /\S/u.test(replyContent);

  const replyActionDisabled =
    loading ||
    !item ||
    item.status !== "listing" ||
    postingReply;

  const openReplyModal = useCallback(() => {
    if (replyActionDisabled) {
      return;
    }

    setReplyError("");
    setIsReplyModalOpen(true);
  }, [replyActionDisabled]);

  const closeReplyModal = useCallback(() => {
    if (postingReply) {
      return;
    }

    setIsReplyModalOpen(false);
    setReplyContent("");
    setReplyError("");
  }, [postingReply]);

  const submitReply = useCallback(async (): Promise<void> => {
    if (
      postingReply ||
      !source ||
      !normalizedResaleId
    ) {
      return;
    }

    if (!/\S/u.test(replyContent)) {
      setReplyError("コメントを入力してください。");
      return;
    }

    if (!item || item.status !== "listing") {
      setReplyError(
        "現在の出品状態ではコメントできません。",
      );
      return;
    }

    setPostingReply(true);
    setReplyError("");

    try {
      const result =
        source === "owner"
          ? await createMyResaleComment({
              resaleId: normalizedResaleId,
              body: replyContent,
            })
          : await createMarketResaleComment({
              resaleId: normalizedResaleId,
              body: replyContent,
            });

      setComments((currentComments) =>
        sortComments([
          ...currentComments.filter(
            (comment) =>
              comment.commentId !==
              result.comment.commentId,
          ),
          result.comment,
        ]),
      );

      setIsReplyModalOpen(false);
      setReplyContent("");
      setReplyError("");
    } catch (caught) {
      setReplyError(
        getErrorMessage(
          caught,
          "コメントの送信に失敗しました。",
        ),
      );
    } finally {
      setPostingReply(false);
    }
  }, [
    item,
    normalizedResaleId,
    postingReply,
    replyContent,
    source,
  ]);

  const handleDeleteComment = useCallback(
    async (comment: ResaleReviewComment): Promise<void> => {
      if (
        !source ||
        !item ||
        item.status !== "listing" ||
        !normalizedResaleId ||
        !viewerAvatarId ||
        comment.avatarId !== viewerAvatarId ||
        comment.isRead ||
        !comment.commentId ||
        deletingCommentId !== ""
      ) {
        return;
      }

      const confirmed = window.confirm(
        "このコメントを削除します。よろしいですか？",
      );

      if (!confirmed) {
        return;
      }

      setDeletingCommentId(comment.commentId);
      setError("");

      try {
        if (source === "owner") {
          await deleteMyResaleComment({
            resaleId: normalizedResaleId,
            commentId: comment.commentId,
          });
        } else {
          await deleteMarketResaleComment({
            resaleId: normalizedResaleId,
            commentId: comment.commentId,
          });
        }

        setComments((currentComments) =>
          currentComments.filter(
            (currentComment) =>
              currentComment.commentId !==
              comment.commentId,
          ),
        );
      } catch (caught) {
        setError(
          getErrorMessage(
            caught,
            "コメントの削除に失敗しました。",
          ),
        );
      } finally {
        setDeletingCommentId("");
      }
    },
    [
      deletingCommentId,
      item,
      normalizedResaleId,
      source,
      viewerAvatarId,
    ],
  );

  return (
    <>
      <Layout
        title={title}
        showBackButton
        onBackButtonClick={onBack}
        showFooter={!isReplyModalOpen}
        mode="mypage"
        mainClassName="chat-detail-page-layout"
        actionButtonLabel="コメント"
        onActionButtonClick={openReplyModal}
        actionButtonDisabled={replyActionDisabled}
        footerProps={{
          variant: "default",
          centerActionLabel: "コメント",
          centerActionDisabled: replyActionDisabled,
          onCenterActionClick: openReplyModal,
        }}
      >
        <section className="page-section content-page-section chat-detail-page">
          {error ? (
            <div
              className="chat-detail-page__error"
              role="alert"
            >
              {error}
            </div>
          ) : null}

          {loading ? (
            <div className="chat-detail-page__state">
              読み込み中...
            </div>
          ) : null}

          {!loading && !item ? (
            <div className="chat-detail-page__empty">
              出品情報が見つかりません。
            </div>
          ) : null}

          {!loading && item ? (
            <div className="chat-detail-page__thread">
              <ResaleThreadHeader
                item={item}
                images={images}
                productMetaItems={productMetaItems}
                onOpenMarketDetail={
                  source === "market" &&
                  item.status === "listing"
                    ? handleOpenMarketDetail
                    : undefined
                }
              />

              <div className="chat-detail-page__reply-section">
                <h3 className="chat-detail-page__section-title">
                  コメント一覧
                </h3>

                {sortedComments.length === 0 ? (
                  <div className="chat-detail-page__no-replies">
                    まだコメントはありません。
                  </div>
                ) : (
                  <div className="chat-detail-page__replies">
                    {sortedComments.map((comment) => {
                      const isMine =
                        Boolean(viewerAvatarId) &&
                        comment.avatarId ===
                          viewerAvatarId;

                      const canDelete =
                        item.status === "listing" &&
                        isMine &&
                        !comment.isRead;

                      return (
                        <ResaleCommentMessage
                          key={comment.commentId}
                          comment={comment}
                          isMine={isMine}
                          canDelete={canDelete}
                          deleting={
                            deletingCommentId ===
                            comment.commentId
                          }
                          onDelete={() => {
                            void handleDeleteComment(
                              comment,
                            );
                          }}
                        />
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </section>
      </Layout>

      <ChatComposerModal
        open={isReplyModalOpen}
        title="コメントする"
        content={replyContent}
        placeholder="コメントを入力"
        error={replyError}
        submitting={postingReply}
        canSubmit={canSubmitReply}
        submitLabel="送信"
        submittingLabel="送信中..."
        onContentChange={setReplyContent}
        onCancel={closeReplyModal}
        onSubmit={() => {
          void submitReply();
        }}
      />
    </>
  );
}

function ResaleThreadHeader({
  item,
  images,
  productMetaItems,
  onOpenMarketDetail,
}: {
  item: ResaleChatItem;
  images: ResaleConditionImage[];
  productMetaItems: ChatMetaItem[];
  onOpenMarketDetail?: () => void;
}) {
  const sellerName =
    item.avatarName ||
    "出品者";

  const productTitle =
    item.productName ||
    item.tokenName ||
    "出品商品";

  return (
    <ChatThreadCard variant="resale">
      <ChatMessageHeader
        name={sellerName}
        icon={item.avatarIcon}
        createdAt={item.createdAt}
        action={
          <span className="chat-detail-page__status">
            {getResaleStatusLabel(item.status)}
          </span>
        }
      />

      <h2 className="chat-detail-page__subject">
        {productTitle}/出品
      </h2>

      <ChatMetaSection
        title="対象商品"
        items={productMetaItems}
      />

      {item.description ? (
        <details className="chat-detail-page__description-accordion">
          <summary className="chat-detail-page__description-summary">
            商品説明
          </summary>

          <div className="chat-detail-page__description-body">
            <p className="chat-detail-page__content">
              {item.description}
            </p>
          </div>
        </details>
      ) : null}

      <ChatImageGrid
        images={images.map((image) => ({
          key: image.id,
          url: image.url,
          alt: "商品状態",
        }))}
      />

      {onOpenMarketDetail ? (
        <div className="chat-detail-page__close-prompt-actions">
          <button
            type="button"
            onClick={onOpenMarketDetail}
          >
            商品ページを見る
          </button>
        </div>
      ) : null}
    </ChatThreadCard>
  );
}

function ResaleCommentMessage({
  comment,
  isMine,
  canDelete,
  deleting,
  onDelete,
}: {
  comment: ResaleReviewComment;
  isMine: boolean;
  canDelete: boolean;
  deleting: boolean;
  onDelete: () => void;
}) {
  const avatarName =
    comment.avatarName ||
    "ユーザー";

  return (
    <ChatMessageBubble
      senderName={avatarName}
      senderIcon={comment.avatarIcon}
      createdAt={comment.createdAt}
      content={comment.body}
      isMine={isMine}
      action={
        canDelete ? (
          <button
            type="button"
            className="chat-detail-page__status"
            disabled={deleting}
            onClick={onDelete}
            style={{
              border: 0,
              cursor: deleting
                ? "not-allowed"
                : "pointer",
            }}
          >
            {deleting
              ? "削除中"
              : "削除"}
          </button>
        ) : undefined
      }
    />
  );
}