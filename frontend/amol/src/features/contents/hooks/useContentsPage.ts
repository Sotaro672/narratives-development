//frontend\amol\src\features\contents\hooks\useContentsPage.ts
import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { MediaGalleryItem } from "../../../components/ui/MediaGallery";
import {
  fetchCurrentAvatarId,
  getApiBaseUrl,
} from "../../catalog/api/catalogApi";
import { useMobilePortrait } from "../../catalog/hooks/useMobilePortrait";
import { useTokenCommentCard } from "../../token-commnet/hooks/useTokenCommentCard";
import { fetchContentsMetadata } from "../api/contentsApi";
import type {
  ContentsMetadata,
  ContentsSearchParams,
} from "../types";

function buildContentsSearchParams(
  searchParams: URLSearchParams
): ContentsSearchParams {
  return {
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
  };
}

export function useContentsPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();
  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);

  const contents = useMemo(
    () => buildContentsSearchParams(searchParams),
    [searchParams]
  );

  const commentCard = useTokenCommentCard({
    tokenBlueprintId: contents.tokenBlueprintId,
  });

  const [metadata, setMetadata] = useState<ContentsMetadata | null>(null);
  const [activeFileIndex, setActiveFileIndex] = useState(0);
  const [currentAvatarId, setCurrentAvatarId] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadingAvatarId, setLoadingAvatarId] = useState(false);
  const [error, setError] = useState("");

  const handleProductNameClick = () => {
    if (!contents.productId) {
      return;
    }

    navigate(`/scan/result?productId=${encodeURIComponent(contents.productId)}`);
  };

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

        if (!isMounted) {
          return;
        }

        setMetadata(result);
        setActiveFileIndex(0);
      } catch (err) {
        if (!isMounted) {
          return;
        }

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
        if (!file.uri) {
          return false;
        }

        if (iconUri && file.uri === iconUri) {
          return false;
        }

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
    if (!hasMediaItems) {
      return;
    }

    setActiveFileIndex((current) =>
      current === 0 ? mediaItems.length - 1 : current - 1
    );
  };

  const handleNextFile = () => {
    if (!hasMediaItems) {
      return;
    }

    setActiveFileIndex((current) =>
      current === mediaItems.length - 1 ? 0 : current + 1
    );
  };

  return {
    contents,
    commentCard,
    metadata,
    mediaItems,
    activeFileIndex,
    setActiveFileIndex,
    currentAvatarId,
    loading,
    loadingAvatarId,
    error,
    tokenName,
    tokenIconUrl,
    pageTitle,
    hasMediaItems,
    isMobilePortrait,
    handleProductNameClick,
    handlePrevFile,
    handleNextFile,
  };
}