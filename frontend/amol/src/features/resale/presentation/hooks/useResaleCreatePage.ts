// frontend/amol/src/features/resale/presentation/hooks/useResaleCreatePage.ts

import {
  useCallback,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
} from "react";
import {
  useLocation,
  useNavigate,
} from "react-router-dom";

import {
  textOrEmpty,
} from "../../../../components/utils/textOrEmpty";

import {
  createResaleListing,
} from "../../api/resaleApi";

import {
  DEFAULT_RESALE_CONDITION,
  normalizeResaleCondition,
} from "../../constants/resaleConditions";

import type {
  ResaleCondition,
} from "../../../shared/types/resale";

import type {
  ResaleCreatePageLocationState,
  ResaleCreateTarget,
} from "../types/resaleCreatePageTypes";

import {
  useResaleConditionMedia,
} from "./useResaleConditionMedia";

const INVALID_FORM_MESSAGE =
  "販売価格と商品状態の写真を入力してください。";

const CREATE_RESALE_ERROR_MESSAGE =
  "出品に失敗しました。時間をおいてもう一度お試しください。";

const RESALE_LOCATION_STATE_KEYS = [
  "mintAddress",
  "productId",
  "brandId",
  "brandName",
  "productName",
  "productBlueprintId",
  "tokenBlueprintId",
  "tokenName",
  "tokenIconUrl",
] as const satisfies readonly (
  keyof ResaleCreatePageLocationState
)[];

function isRecord(
  value: unknown,
): value is Record<string, unknown> {
  return (
    typeof value === "object" &&
    value !== null &&
    !Array.isArray(value)
  );
}

function parseLocationState(
  value: unknown,
): ResaleCreatePageLocationState {
  if (!isRecord(value)) {
    return {};
  }

  const result:
    ResaleCreatePageLocationState = {};

  RESALE_LOCATION_STATE_KEYS.forEach(
    (key) => {
      const fieldValue =
        value[key];

      if (
        typeof fieldValue ===
        "string"
      ) {
        result[key] =
          fieldValue;
      }
    },
  );

  return result;
}

function createResaleTarget(
  state:
    ResaleCreatePageLocationState,
): ResaleCreateTarget {
  return {
    mintAddress:
      textOrEmpty(
        state.mintAddress,
      ),
    productId:
      textOrEmpty(
        state.productId,
      ),
    brandId:
      textOrEmpty(
        state.brandId,
      ),
    brandName:
      textOrEmpty(
        state.brandName,
      ),
    productName:
      textOrEmpty(
        state.productName,
      ),
    productBlueprintId:
      textOrEmpty(
        state.productBlueprintId,
      ),
    tokenBlueprintId:
      textOrEmpty(
        state.tokenBlueprintId,
      ),
    tokenName:
      textOrEmpty(
        state.tokenName,
      ),
    tokenIconUrl:
      textOrEmpty(
        state.tokenIconUrl,
      ),
  };
}

function normalizePriceInput(
  value: string,
): string {
  return value.replace(
    /[^\d]/g,
    "",
  );
}

function formatPriceInput(
  value: string,
): string {
  const digits =
    normalizePriceInput(
      value,
    );

  if (!digits) {
    return "";
  }

  return Number(
    digits,
  ).toLocaleString(
    "ja-JP",
  );
}

export function useResaleCreatePage() {
  const navigate =
    useNavigate();

  const location =
    useLocation();

  const [
    price,
    setPrice,
  ] = useState("");

  const [
    condition,
    setCondition,
  ] = useState<ResaleCondition>(
    DEFAULT_RESALE_CONDITION,
  );

  const [
    description,
    setDescription,
  ] = useState("");

  const [
    errorMessage,
    setErrorMessage,
  ] = useState("");

  const [
    isSubmitting,
    setIsSubmitting,
  ] = useState(false);

  const isSubmittingRef =
    useRef(false);

  const {
    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,
    hasConditionMedia,
    handleConditionMediaSelected:
      selectConditionMedia,
    handleRemoveConditionMedia:
      removeConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
    clearConditionMedia,
  } = useResaleConditionMedia();

  const locationState =
    useMemo(
      () =>
        parseLocationState(
          location.state,
        ),
      [location.state],
    );

  const target =
    useMemo(
      () =>
        createResaleTarget(
          locationState,
        ),
      [locationState],
    );

  const formattedPrice =
    useMemo(
      () =>
        formatPriceInput(
          price,
        ),
      [price],
    );

  const hasRequiredListingTarget =
    Boolean(
      target.productId,
    ) &&
    Boolean(
      target.tokenBlueprintId,
    );

  const canSubmit =
    hasRequiredListingTarget &&
    Boolean(price) &&
    hasConditionMedia;

  const handlePriceChange =
    useCallback(
      (
        value: string,
      ) => {
        setPrice(
          normalizePriceInput(
            value,
          ),
        );

        setErrorMessage("");
      },
      [],
    );

  const handleConditionChange =
    useCallback(
      (
        value: string,
      ) => {
        setCondition(
          normalizeResaleCondition(
            value,
          ),
        );

        setErrorMessage("");
      },
      [],
    );

  const handleDescriptionChange =
    useCallback(
      (
        value: string,
      ) => {
        setDescription(
          value,
        );

        setErrorMessage("");
      },
      [],
    );

  const handleConditionMediaSelected =
    useCallback(
      (
        event:
          ChangeEvent<HTMLInputElement>,
      ) => {
        selectConditionMedia(
          event,
        );

        setErrorMessage("");
      },
      [
        selectConditionMedia,
      ],
    );

  const handleRemoveConditionMedia =
    useCallback(
      (
        id: string,
      ) => {
        removeConditionMedia(
          id,
        );

        setErrorMessage("");
      },
      [
        removeConditionMedia,
      ],
    );

  const handleBackToWallet =
    useCallback(() => {
      navigate(
        "/wallet",
      );
    }, [
      navigate,
    ]);

  const handleSubmit =
    useCallback(
      async (): Promise<void> => {
        if (
          isSubmittingRef.current
        ) {
          return;
        }

        if (!canSubmit) {
          setErrorMessage(
            INVALID_FORM_MESSAGE,
          );

          return;
        }

        isSubmittingRef.current =
          true;

        setIsSubmitting(true);
        setErrorMessage("");

        try {
          const created =
            await createResaleListing({
              mintAddress:
                target.mintAddress,
              tokenBlueprintId:
                target.tokenBlueprintId,
              productId:
                target.productId,
              brandId:
                target.brandId,
              productBlueprintId:
                target.productBlueprintId,
              price:
                Number(price),
              condition,
              description,
              conditionImages:
                conditionMediaItems.map(
                  (item) =>
                    item.file,
                ),
            });

          navigate(
            "/wallet",
            {
              replace: true,
              state: {
                resaleCreated:
                  true,
                resaleId:
                  created?.id,
              },
            },
          );
        } catch (error) {
          setErrorMessage(
            error instanceof Error
              ? error.message
              : CREATE_RESALE_ERROR_MESSAGE,
          );
        } finally {
          isSubmittingRef.current =
            false;

          setIsSubmitting(false);
        }
      },
      [
        canSubmit,
        condition,
        conditionMediaItems,
        description,
        navigate,
        price,
        target,
      ],
    );

  return {
    target,

    price,
    formattedPrice,
    condition,
    description,

    conditionMediaItems,
    conditionMediaCurrentIndex,
    conditionMediaInputRef,
    conditionMediaCarouselRef,

    hasRequiredListingTarget,
    hasConditionMedia,
    canSubmit,
    isSubmitting,
    errorMessage,

    submitButtonLabel:
      isSubmitting
        ? "出品中..."
        : "出品する",

    handlePriceChange,
    handleConditionChange,
    handleDescriptionChange,

    handleConditionMediaSelected,
    handleRemoveConditionMedia,
    handleConditionMediaCarouselScroll,
    handleMoveToConditionMediaSlide,
    clearConditionMedia,

    handleBackToWallet,
    handleSubmit,
  };
}