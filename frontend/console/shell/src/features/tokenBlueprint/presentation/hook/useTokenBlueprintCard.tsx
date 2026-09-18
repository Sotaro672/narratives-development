// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCard.tsx

import * as React from "react";

import { validateImageForStorage } from "../../../../shared/storage/imageStoragePolicy";
import type { IconCropPosition } from "../../../../shared/types/iconCrop";
import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";
import { cropIconImage } from "../../../../shared/util/cropIconImage";
import { useBrandSelection } from "../../../brand/presentation/hook/useBrandSelection";
import type {
  TokenBlueprintCardHandlers,
  TokenBlueprintCardViewModel,
} from "../components/tokenBlueprintCard";

const INITIAL_ICON_CROP_POSITION: IconCropPosition = {
  x: 0,
  y: 0,
};

const INITIAL_ICON_CROP_SCALE = 1;

/**
 * TokenBlueprintCard用のロジックフック。
 *
 * ブランド候補取得・ブランド選択はuseBrandSelectionを正とする。
 *
 * 仕様:
 * - ブランド候補はisActive=trueのみ
 * - mintedはbooleanとして扱う
 * - minted=trueでもトークンアイコンは編集できる
 * - minted=trueの場合、トークン名・シンボル・ブランドは変更できない
 * - APIスキーマはname・brandNameを正とする
 * - アイコン画像の検証はshared/storage/imageStoragePolicy.tsを正とする
 * - TokenIconの切り抜き処理はshared/util/cropIconImage.tsを正とする
 */
export function useTokenBlueprintCard(params: {
  initialTokenBlueprint?: Partial<TokenBlueprint>;
  initialBurnAt?: string;
  initialIconUrl?: string;
  initialEditMode?: boolean;
}) {
  const tokenBlueprint = params.initialTokenBlueprint ?? {};

  const pickBrandName = React.useCallback(
    (source: Partial<TokenBlueprint>): string => {
      return String(source.brandName ?? "").trim();
    },
    [],
  );

  const pickString = React.useCallback((value: unknown): string => {
    return String(value ?? "").trim();
  }, []);

  const [id, setId] = React.useState(pickString(tokenBlueprint.id));
  const [name, setName] = React.useState(pickString(tokenBlueprint.name));
  const [symbol, setSymbol] = React.useState(pickString(tokenBlueprint.symbol));
  const [description, setDescription] = React.useState(pickString(tokenBlueprint.description));
  const [burnAt, setBurnAt] = React.useState(params.initialBurnAt ?? "");
  const [minted, setMinted] = React.useState<boolean>(tokenBlueprint.minted ?? false);
  const [remoteIconUrl, setRemoteIconUrl] = React.useState(params.initialIconUrl ?? "");
  const [localPreviewUrl, setLocalPreviewUrl] = React.useState("");
  const [selectedIconFile, setSelectedIconFile] = React.useState<File | null>(null);
  const [iconCropPosition, setIconCropPosition] = React.useState<IconCropPosition>(INITIAL_ICON_CROP_POSITION);
  const [iconCropScale, setIconCropScale] = React.useState(INITIAL_ICON_CROP_SCALE);
  const [iconCropViewportSize, setIconCropViewportSize] = React.useState(0);
  const [isEditMode, setIsEditMode] = React.useState(params.initialEditMode ?? false);

  const {
    brandId,
    brandName,
    brandOptions,
    selectBrand,
  } = useBrandSelection({
    initialBrandId: pickString(tokenBlueprint.brandId),
    initialBrandName: pickBrandName(tokenBlueprint),
  });

  const initialRef = React.useRef<Partial<TokenBlueprint> | null>(tokenBlueprint);
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);
  const iconInputRef = React.useRef<HTMLInputElement | null>(null);
  const localPreviewUrlRef = React.useRef("");

  const canEditIcon = Boolean(isEditMode || minted);
  const isIdentityLocked = Boolean(minted);

  const resetIconCrop = React.useCallback(() => {
    setIconCropPosition(INITIAL_ICON_CROP_POSITION);
    setIconCropScale(INITIAL_ICON_CROP_SCALE);
    setIconCropViewportSize(0);
  }, []);

  const clearLocalPreview = React.useCallback(() => {
    const currentUrl = localPreviewUrlRef.current;

    if (currentUrl) {
      URL.revokeObjectURL(currentUrl);
      localPreviewUrlRef.current = "";
      setLocalPreviewUrl("");
    }
  }, []);

  React.useEffect(() => {
    const source = params.initialTokenBlueprint;

    if (!source) {
      return;
    }

    initialRef.current = source;

    if (isEditMode) {
      return;
    }

    setId(pickString(source.id));
    setName(pickString(source.name));
    setSymbol(pickString(source.symbol));
    selectBrand(pickString(source.brandId));
    setDescription(pickString(source.description));
    setMinted(source.minted ?? false);
    setBurnAt(params.initialBurnAt ?? "");
    setSelectedIconFile(null);
    resetIconCrop();
    clearLocalPreview();
  }, [
    params.initialTokenBlueprint,
    params.initialBurnAt,
    isEditMode,
    pickString,
    selectBrand,
    resetIconCrop,
    clearLocalPreview,
  ]);

  React.useEffect(() => {
    if (isEditMode) {
      return;
    }

    setRemoteIconUrl(params.initialIconUrl ?? "");
  }, [params.initialIconUrl, isEditMode]);

  React.useEffect(() => {
    const element = descriptionRef.current;

    if (!element) {
      return;
    }

    element.style.height = "auto";
    element.style.height = `${element.scrollHeight}px`;
  }, [description]);

  React.useEffect(() => {
    return () => {
      const currentUrl = localPreviewUrlRef.current;

      if (currentUrl) {
        URL.revokeObjectURL(currentUrl);
        localPreviewUrlRef.current = "";
      }
    };
  }, []);

  const requestPickIconFile = React.useCallback(() => {
    if (!canEditIcon) {
      return;
    }

    iconInputRef.current?.click();
  }, [canEditIcon]);

  const onIconInputChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      if (!canEditIcon) {
        event.target.value = "";
        return;
      }

      const file = event.target.files?.[0] ?? null;
      event.target.value = "";

      if (!file) {
        return;
      }

      const validation = validateImageForStorage(file, "tokenBlueprintIcon");

      if (!validation.valid) {
        setSelectedIconFile(null);
        resetIconCrop();
        clearLocalPreview();
        window.alert(validation.reason);
        return;
      }

      clearLocalPreview();
      resetIconCrop();
      setSelectedIconFile(file);

      const previewUrl = URL.createObjectURL(file);
      localPreviewUrlRef.current = previewUrl;
      setLocalPreviewUrl(previewUrl);
    },
    [canEditIcon, clearLocalPreview, resetIconCrop],
  );

  const buildIconFileForUpload = React.useCallback(async (): Promise<File | null> => {
    if (!selectedIconFile) {
      return null;
    }

    if (iconCropViewportSize <= 0) {
      throw new Error("アイコン画像の切り抜き領域を取得できませんでした。画像を選択し直してください。");
    }

    const croppedFile = await cropIconImage({
      file: selectedIconFile,
      position: iconCropPosition,
      scale: iconCropScale,
      viewportSize: iconCropViewportSize,
    });

    const validation = validateImageForStorage(croppedFile, "tokenBlueprintIcon");

    if (!validation.valid) {
      throw new Error(validation.reason);
    }

    return croppedFile;
  }, [
    selectedIconFile,
    iconCropPosition,
    iconCropScale,
    iconCropViewportSize,
  ]);

  const shownIconUrl = localPreviewUrl || remoteIconUrl;

  const vm: TokenBlueprintCardViewModel = {
    id,
    name,
    symbol,
    brandId,
    brandName,
    description,
    iconUrl: shownIconUrl,
    minted,
    isEditMode,
    brandOptions,
    iconFile: selectedIconFile,
    iconCropPosition,
    iconCropScale,
  };

  const handlers: TokenBlueprintCardHandlers = {
    onChangeName: (value: string) => {
      if (isIdentityLocked) {
        return;
      }

      setName(value);
    },

    onChangeSymbol: (value: string) => {
      if (isIdentityLocked) {
        return;
      }

      setSymbol(value.toUpperCase());
    },

    onChangeBrand: (nextBrandId: string, _nextBrandName: string) => {
      if (isIdentityLocked) {
        return;
      }

      selectBrand(nextBrandId);
    },

    onChangeDescription: (value: string) => {
      setDescription(value);
    },

    iconInputRef,
    descriptionRef,
    onRequestPickIconFile: requestPickIconFile,
    onIconInputChange,

    onIconCropPositionChange: (position: IconCropPosition) => {
      setIconCropPosition(position);
    },

    onIconCropScaleChange: (scale: number) => {
      setIconCropScale(scale);
    },

    onIconCropViewportSizeChange: (size: number) => {
      setIconCropViewportSize(size);
    },

    onClearLocalIconFile: () => {
      setSelectedIconFile(null);
      resetIconCrop();
      clearLocalPreview();
    },

    onToggleEditMode: () => {
      setIsEditMode((current) => !current);
    },

    setEditMode: (edit: boolean) => {
      setIsEditMode(edit);
    },

    reset: () => {
      const source = initialRef.current;

      if (!source) {
        return;
      }

      setId(pickString(source.id));
      setName(pickString(source.name));
      setSymbol(pickString(source.symbol));
      selectBrand(pickString(source.brandId));
      setDescription(pickString(source.description));
      setMinted(source.minted ?? false);
      setBurnAt(params.initialBurnAt ?? "");
      setRemoteIconUrl(params.initialIconUrl ?? "");
      setSelectedIconFile(null);
      resetIconCrop();
      clearLocalPreview();
    },
  };

  return {
    vm,
    handlers,
    selectedIconFile,
    burnAt,
    canEditIcon,
    buildIconFileForUpload,
  };
}