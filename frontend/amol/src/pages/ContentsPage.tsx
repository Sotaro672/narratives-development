// frontend/amol/src/pages/ContentsPage.tsx
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { getAuth } from "firebase/auth";

import "../styles/page-layout.css";
import "../styles/contents-page.css";

import Layout from "../components/layout/Layout";
import MediaGallery, {
  type MediaGalleryItem,
} from "../components/ui/MediaGallery";
import Tab from "../components/ui/Tab";
import {
  fetchCurrentAvatarId,
  getApiBaseUrl,
} from "../features/catalog/api/catalogApi";
import { useMobilePortrait } from "../features/catalog/hooks/useMobilePortrait";
import TokenCommentCard from "../features/token-commnet/components/TokenCommentCard";
import TokenReviewAggregateCard from "../features/token-commnet/components/TokenReviewAggregateCard";
import { useTokenCommentCard } from "../features/token-commnet/hooks/useTokenCommentCard";

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;

type ContentsMetadataFile = {
  name: string;
  type: string;
  uri: string;
};

type ContentsMetadata = {
  name: string;
  symbol: string;
  description: string;
  image: string;
  createdAt: string;
  files: ContentsMetadataFile[];
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function getString(value: Record<string, unknown>, key: string): string {
  const raw = value[key];
  return typeof raw === "string" ? raw : "";
}

function parseMetadataFile(value: unknown): ContentsMetadataFile | null {
  if (!isRecord(value)) {
    return null;
  }

  const uri = getString(value, "uri");
  const type = getString(value, "type");
  const name = getString(value, "name");

  if (!uri) {
    return null;
  }

  return {
    name,
    type,
    uri,
  };
}

function parseContentsMetadata(value: unknown): ContentsMetadata | null {
  if (!isRecord(value)) {
    return null;
  }

  const properties = isRecord(value.properties) ? value.properties : null;
  const filesRaw =
    properties && Array.isArray(properties.files) ? properties.files : [];

  const files = filesRaw
    .map(parseMetadataFile)
    .filter((file): file is ContentsMetadataFile => file !== null);

  return {
    name: getString(value, "name"),
    symbol: getString(value, "symbol"),
    description: getString(value, "description"),
    image: getString(value, "image"),
    createdAt: getString(value, "created_at"),
    files,
  };
}

function normalizeBackendUrl(backendUrl: string): string {
  return backendUrl.replace(/\/+$/, "");
}

async function fetchContentsMetadata(
  metadataUri: string
): Promise<ContentsMetadata | null> {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_API_BASE_URL is not configured.");
  }

  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログインが必要です。");
  }

  const idToken = await user.getIdToken();
  const baseUrl = normalizeBackendUrl(BACKEND_BASE_URL);
  const url = new URL(`${baseUrl}/mall/me/wallets/metadata/proxy`);

  url.searchParams.set("url", metadataUri);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`metadata fetch failed: ${response.status} ${body}`);
  }

  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    throw new Error("metadata API が JSON 以外を返しました。");
  }

  const body: unknown = await response.json();

  return parseContentsMetadata(body);
}

export default function ContentsPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);

  const contents = useMemo(
    () => ({
      mintAddress: searchParams.get("mintAddress") || "",
      productId: searchParams.get("productId") || "",
      brandId: searchParams.get("brandId") || "",
      brandName: searchParams.get("brandName") || "",
      productName: searchParams.get("productName") || "",
      productBlueprintId: searchParams.get("productBlueprintId") || "",
      tokenBlueprintId: searchParams.get("tokenBlueprintId") || "",
      metadataUri: searchParams.get("metadataUri") || "",
      tokenName: searchParams.get("tokenName") || "",
      tokenIconUrl: searchParams.get("tokenIconUrl") || "",
    }),
    [searchParams]
  );

  const handleProductNameClick = () => {
    if (!contents.productId) {
      return;
    }

    navigate(`/scan/result?productId=${encodeURIComponent(contents.productId)}`);
  };

  const commentCard = useTokenCommentCard({
    tokenBlueprintId: contents.tokenBlueprintId,
  });

  const [metadata, setMetadata] = useState<ContentsMetadata | null>(null);
  const [activeFileIndex, setActiveFileIndex] = useState(0);
  const [currentAvatarId, setCurrentAvatarId] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadingAvatarId, setLoadingAvatarId] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    let isMounted = true;

    const loadCurrentAvatarId = async () => {
      setLoadingAvatarId(true);

      try {
        const avatarId = await fetchCurrentAvatarId(apiBaseUrl);

        if (!isMounted) {
          return;
        }

        setCurrentAvatarId(avatarId);
      } catch {
        if (!isMounted) {
          return;
        }

        setCurrentAvatarId("");
      } finally {
        if (isMounted) {
          setLoadingAvatarId(false);
        }
      }
    };

    void loadCurrentAvatarId();

    return () => {
      isMounted = false;
    };
  }, [apiBaseUrl]);

  useEffect(() => {
    if (!contents.metadataUri) {
      setMetadata(null);
      setError("");
      return;
    }

    let isMounted = true;

    const load = async () => {
      setLoading(true);
      setError("");

      try {
        const result = await fetchContentsMetadata(contents.metadataUri);

        if (!isMounted) return;

        setMetadata(result);
        setActiveFileIndex(0);
      } catch (err) {
        if (!isMounted) return;

        setMetadata(null);
        setActiveFileIndex(0);
        setError(
          err instanceof Error
            ? err.message
            : "トークンコンテンツの取得に失敗しました。"
        );
      } finally {
        if (isMounted) {
          setLoading(false);
        }
      }
    };

    void load();

    return () => {
      isMounted = false;
    };
  }, [contents.metadataUri]);

  const tokenName = metadata?.name || contents.tokenName;
  const tokenIconUrl = metadata?.image || contents.tokenIconUrl;
  const pageTitle = tokenName || "トークン詳細";

  const mediaItems = useMemo<MediaGalleryItem[]>(() => {
    const iconUri = metadata?.image || contents.tokenIconUrl;

    return (metadata?.files || [])
      .filter((file) => {
        if (!file.uri) return false;
        if (iconUri && file.uri === iconUri) return false;
        return true;
      })
      .map((file, index) => ({
        id: `${index}-${file.uri}`,
        url: file.uri,
        fileName: file.name,
        type: file.type,
      }));
  }, [metadata?.files, metadata?.image, contents.tokenIconUrl]);

  useEffect(() => {
    if (activeFileIndex >= mediaItems.length) {
      setActiveFileIndex(0);
    }
  }, [activeFileIndex, mediaItems.length]);

  const hasMediaItems = mediaItems.length > 0;

  const handlePrevFile = () => {
    if (!hasMediaItems) return;

    setActiveFileIndex((current) =>
      current === 0 ? mediaItems.length - 1 : current - 1
    );
  };

  const handleNextFile = () => {
    if (!hasMediaItems) return;

    setActiveFileIndex((current) =>
      current === mediaItems.length - 1 ? 0 : current + 1
    );
  };

  return (
    <Layout
      title={pageTitle}
      mode="mypage"
      showBackButton
      backTo="/wallet"
      hideHamburgerMenu
      showFooter
      disableFooterPaddingOnDesktop
      footerProps={
        isMobilePortrait
          ? {
              variant: "commentAction",
              value: commentCard.commentBody,
              placeholder: "コメントを書く…",
              buttonLabel: commentCard.posting ? "投稿中" : "投稿",
              disabled:
                commentCard.posting ||
                loading ||
                !contents.tokenBlueprintId ||
                !commentCard.commentBody.trim(),
              posting: commentCard.posting,
              onChange: commentCard.setCommentBody,
              onSubmit: commentCard.postComment,
            }
          : { variant: "default" }
      }
    >
      <section className="split-page contents-page">
        <div className="split-page-content contents-page-content">
          <div className="split-page-left contents-page-media-area">
            {loading ? (
              <p className="contents-page-card__message">読み込み中です...</p>
            ) : null}

            {!loading && error ? (
              <p className="contents-page-card__error">{error}</p>
            ) : null}

            {!loading && !error && !contents.metadataUri ? (
              <p className="contents-page-card__error">
                metadataUri が指定されていません。
              </p>
            ) : null}

            {!loading && !error && contents.metadataUri && !hasMediaItems ? (
              <p className="contents-page-card__message">
                表示できるコンテンツはまだありません。
              </p>
            ) : null}

            {!loading && !error && hasMediaItems ? (
              <MediaGallery
                items={mediaItems}
                activeIndex={activeFileIndex}
                altFallback={tokenName || "トークンコンテンツ"}
                className="contents-page-media-gallery"
                onPrev={handlePrevFile}
                onNext={handleNextFile}
                onSelect={setActiveFileIndex}
              />
            ) : null}
          </div>

          <div className="split-page-right contents-page-detail">
            <div className="contents-page-card">
              <div className="contents-page-card__header">
                <div className="contents-page-card__icon-wrap">
                  {tokenIconUrl ? (
                    <img
                      src={tokenIconUrl}
                      alt={tokenName || "トークンアイコン"}
                      className="contents-page-card__icon"
                    />
                  ) : (
                    <div className="contents-page-card__icon contents-page-card__icon--fallback">
                      ◎
                    </div>
                  )}
                </div>

                <div className="contents-page-card__meta">
                  <p className="contents-page-card__title">
                    {tokenName || "名称未設定のトークン"}
                  </p>

                  {contents.productName ? (
                    <Tab
                      className="contents-page-card__product-name"
                      onClick={handleProductNameClick}
                      disabled={!contents.productId}
                    >
                      {contents.productName}
                    </Tab>
                  ) : null}

                  {contents.brandName ? (
                    <p className="contents-page-card__brand-name">
                      {contents.brandName}
                    </p>
                  ) : null}
                </div>
              </div>

              <TokenReviewAggregateCard
                tokenBlueprintId={contents.tokenBlueprintId}
                productId={contents.productId}
                currentAvatarId={currentAvatarId}
                shareTitle={tokenName || "トークン詳細"}
                shareText={contents.productName || contents.brandName || ""}
                shareUrl={window.location.href}
              />

              {loadingAvatarId ? (
                <p className="contents-page-card__message">
                  アバター情報を確認しています...
                </p>
              ) : null}
            </div>

            <TokenCommentCard
              tokenBlueprintId={contents.tokenBlueprintId}
              loading={loading}
              hideCommentForm={isMobilePortrait}
              commentTree={commentCard.commentTree}
              commentsLoading={commentCard.commentsLoading}
              commentsError={commentCard.commentsError}
              posting={commentCard.posting}
              commentBody={commentCard.commentBody}
              expandedIds={commentCard.expandedIds}
              replyingCommentId={commentCard.replyingCommentId}
              replyBody={commentCard.replyBody}
              replyPosting={commentCard.replyPosting}
              onCommentBodyChange={commentCard.setCommentBody}
              onReplyBodyChange={commentCard.setReplyBody}
              onRefreshComments={commentCard.refreshComments}
              onPostComment={commentCard.postComment}
              onToggleExpanded={commentCard.toggleExpanded}
              onLikeComment={commentCard.likeComment}
              onDislikeComment={commentCard.dislikeComment}
              onStartReply={commentCard.startReply}
              onCancelReply={commentCard.cancelReply}
              onSubmitReply={commentCard.submitReply}
            />
          </div>
        </div>
      </section>
    </Layout>
  );
}