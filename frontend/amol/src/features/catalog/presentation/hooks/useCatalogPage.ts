// frontend/amol/src/features/catalog/presentation/hooks/useCatalogPage.ts

import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { MediaGalleryItem } from "../../../../components/ui/MediaGallery";
import { useMobilePortrait } from "../../../../components/hooks/useMobilePortrait";
import { getApiBaseUrl } from "../../../../lib/apiBaseUrl";

import { addSelectedCatalogItemToCart } from "../../application/catalogCartUsecase";
import { loadCatalogPage } from "../../application/catalogPageLoader";
import { createCatalogPageViewModel } from "../../application/catalogPageViewModelFactory";

import type { CatalogResponse } from "../../../shared/types/catalog";
import type { ProductBlueprintReviewPage } from "../../../shared/types/review";

export function useCatalogPage() {
  const navigate = useNavigate();
  const { listId } = useParams();

  const [catalog, setCatalog] = useState<CatalogResponse | null>(null);
  const [reviews, setReviews] = useState<ProductBlueprintReviewPage | null>(null);
  const [isLoadingCatalog, setIsLoadingCatalog] = useState(true);
  const [isAddingToCart, setIsAddingToCart] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [reviewErrorMessage, setReviewErrorMessage] = useState("");
  const [cartErrorMessage, setCartErrorMessage] = useState("");
  const [activeImageIndex, setActiveImageIndex] = useState(0);
  const [selectedColorKey, setSelectedColorKey] = useState("");
  const [selectedSize, setSelectedSize] = useState("");
  const [selectedModelId, setSelectedModelId] = useState("");

  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);
  const isMobilePortrait = useMobilePortrait();

  const viewModel = useMemo(() => {
    return createCatalogPageViewModel({
      catalog,
      reviews,
      selectedColorKey,
      selectedSize,
      selectedModelId,
      activeImageIndex,
      isAddingToCart,
    });
  }, [
    catalog,
    reviews,
    selectedColorKey,
    selectedSize,
    selectedModelId,
    activeImageIndex,
    isAddingToCart,
  ]);

  const galleryItems = useMemo<MediaGalleryItem[]>(
    () =>
      viewModel.catalogImages
        .filter((image) => Boolean(image.url))
        .map((image) => ({
          id: image.id,
          url: image.url,
          fileName: image.fileName,
        })),
    [viewModel.catalogImages],
  );

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!listId) {
        setCatalog(null);
        setReviews(null);
        setErrorMessage("listIdが見つかりません。");
        setReviewErrorMessage("");
        setIsLoadingCatalog(false);
        return;
      }

      setIsLoadingCatalog(true);
      setErrorMessage("");
      setReviewErrorMessage("");
      setCartErrorMessage("");
      setCatalog(null);
      setReviews(null);
      setActiveImageIndex(0);
      setSelectedColorKey("");
      setSelectedSize("");
      setSelectedModelId("");

      try {
        const result = await loadCatalogPage({ apiBaseUrl, listId });
        if (cancelled) return;

        setCatalog(result.catalog);
        setReviews(result.reviews);
        setReviewErrorMessage(result.reviewErrorMessage);
      } catch (error) {
        if (cancelled) return;

        setCatalog(null);
        setReviews(null);
        setReviewErrorMessage("");
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "カタログ詳細の取得中にエラーが発生しました。",
        );
      } finally {
        if (!cancelled) setIsLoadingCatalog(false);
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl, listId]);

  useEffect(() => {
    if (viewModel.isAlcoholCatalog) return;

    if (viewModel.colorOptions.length === 1 && !selectedColorKey) {
      setSelectedColorKey(viewModel.colorOptions[0].key);
      return;
    }

    if (
      selectedColorKey &&
      !viewModel.colorOptions.some((option) => option.key === selectedColorKey)
    ) {
      setSelectedColorKey("");
    }
  }, [selectedColorKey, viewModel.colorOptions, viewModel.isAlcoholCatalog]);

  useEffect(() => {
    if (viewModel.isAlcoholCatalog) return;

    if (viewModel.sizeOptions.length === 1 && !selectedSize) {
      setSelectedSize(viewModel.sizeOptions[0]);
      return;
    }

    if (selectedSize && !viewModel.sizeOptions.includes(selectedSize)) {
      setSelectedSize("");
    }
  }, [selectedSize, viewModel.sizeOptions, viewModel.isAlcoholCatalog]);

  useEffect(() => {
    if (!viewModel.isAlcoholCatalog) {
      if (selectedModelId) setSelectedModelId("");
      return;
    }

    if (viewModel.alcoholOptions.length === 1 && !selectedModelId) {
      setSelectedModelId(viewModel.alcoholOptions[0].modelId);
      return;
    }

    if (
      selectedModelId &&
      !viewModel.alcoholOptions.some((option) => option.modelId === selectedModelId)
    ) {
      setSelectedModelId("");
    }
  }, [selectedModelId, viewModel.alcoholOptions, viewModel.isAlcoholCatalog]);

  useEffect(() => {
    if (activeImageIndex >= galleryItems.length) {
      setActiveImageIndex(0);
    }
  }, [activeImageIndex, galleryItems.length]);

  function handlePrevImage() {
    if (galleryItems.length === 0) return;

    setActiveImageIndex((current) =>
      current === 0 ? galleryItems.length - 1 : current - 1,
    );
  }

  function handleNextImage() {
    if (galleryItems.length === 0) return;

    setActiveImageIndex((current) =>
      current === galleryItems.length - 1 ? 0 : current + 1,
    );
  }

  function handleSelectImage(index: number) {
    if (index < 0 || index >= galleryItems.length) return;
    setActiveImageIndex(index);
  }

  function handleSelectColor(colorKey: string) {
    setSelectedColorKey(colorKey);
    setSelectedSize("");
    setCartErrorMessage("");
  }

  function handleSelectSize(size: string) {
    setSelectedSize(size);
    setCartErrorMessage("");
  }

  function handleSelectModel(modelId: string) {
    setSelectedModelId(modelId);
    setCartErrorMessage("");
  }

  function handleBrandClick() {
    const brandId = catalog?.productBlueprint.brandId.trim();
    if (!brandId) return;

    navigate(`/brands/${encodeURIComponent(brandId)}`);
  }

  function handleAvatarClick(avatarId: string) {
    const normalizedAvatarId = avatarId.trim();
    if (!normalizedAvatarId) return;

    navigate(`/avatars/${encodeURIComponent(normalizedAvatarId)}`);
  }

  async function handleAddToCart() {
    setIsAddingToCart(true);
    setCartErrorMessage("");

    try {
      await addSelectedCatalogItemToCart({
        apiBaseUrl,
        catalog,
        selectedModel: viewModel.selectedModel,
        hasSelectedModelStock: viewModel.hasSelectedModelStock,
        isAlcoholCatalog: viewModel.isAlcoholCatalog,
      });

      navigate("/cart");
    } catch (error) {
      setCartErrorMessage(
        error instanceof Error
          ? error.message
          : "カートへの追加中にエラーが発生しました。",
      );
    } finally {
      setIsAddingToCart(false);
    }
  }

  return {
    catalog,
    isLoadingCatalog,
    isAddingToCart,
    errorMessage,
    reviewErrorMessage,
    cartErrorMessage,
    selectedColorKey,
    selectedSize,
    selectedModelId,
    activeImageIndex,
    galleryItems,
    isMobilePortrait,
    ...viewModel,
    handlePrevImage,
    handleNextImage,
    handleSelectImage,
    handleSelectColor,
    handleSelectSize,
    handleSelectModel,
    handleBrandClick,
    handleAvatarClick,
    handleAddToCart,
  };
}