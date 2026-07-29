// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintCard.tsx

import * as React from "react";

import type { TokenBlueprint } from "../../domain/tokenBlueprint";

import type {
  TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlers,
} from "../components/tokenBlueprintCard";

import { loadBrandsForCompany } from "../../application/tokenBlueprintCreateService";

/**
 * TokenBlueprintCard用のロジックフック
 * - UI状態管理
 * - アイコンファイルの選択
 * - アイコンのローカルプレビュー
 *
 * 仕様:
 * - mintedはbooleanとして扱う
 * - minted=trueでもトークンアイコンは編集できる
 * - minted=trueの場合、トークン名・シンボル・ブランドは変更できない
 * - APIスキーマはname・brandNameを正とする
 * - ブランド名は/brandsの一覧レスポンスitems[].nameまたは
 *   TokenBlueprint.brandNameを正とする
 * - brandIdからの個別名前解決は行わない
 */
export function useTokenBlueprintCard(params: {
  initialTokenBlueprint?: Partial<TokenBlueprint> & {
    brandName?: string;
  };
  initialBurnAt?: string;
  initialIconUrl?: string;
  initialEditMode?: boolean;
}) {
  const tokenBlueprint =
    params.initialTokenBlueprint ?? {};

  const pickBrandName = React.useCallback(
    (source: unknown): string => {
      const value = source as {
        brandName?: unknown;
      };

      return String(
        value?.brandName ?? "",
      ).trim();
    },
    [],
  );

  const pickString = React.useCallback(
    (value: unknown): string => {
      return String(value ?? "").trim();
    },
    [],
  );

  const [id, setId] = React.useState(
    pickString(tokenBlueprint.id),
  );

  const [name, setName] = React.useState(
    pickString(tokenBlueprint.name),
  );

  const [symbol, setSymbol] = React.useState(
    pickString(tokenBlueprint.symbol),
  );

  const [brandId, setBrandId] = React.useState(
    pickString(tokenBlueprint.brandId),
  );

  const [brandName, setBrandName] = React.useState(
    pickBrandName(tokenBlueprint),
  );

  const [description, setDescription] =
    React.useState(
      pickString(tokenBlueprint.description),
    );

  const [burnAt, setBurnAt] = React.useState(
    params.initialBurnAt ?? "",
  );

  const [minted, setMinted] =
    React.useState<boolean>(
      typeof tokenBlueprint.minted === "boolean"
        ? tokenBlueprint.minted
        : false,
    );

  const [remoteIconUrl, setRemoteIconUrl] =
    React.useState(
      params.initialIconUrl ?? "",
    );

  const [localPreviewUrl, setLocalPreviewUrl] =
    React.useState<string>("");

  const [
    selectedIconFile,
    setSelectedIconFile,
  ] = React.useState<File | null>(null);

  const [isEditMode, setIsEditMode] =
    React.useState(
      params.initialEditMode ?? false,
    );

  const [brandOptions, setBrandOptions] =
    React.useState<
      {
        id: string;
        name: string;
      }[]
    >([]);

  const initialRef = React.useRef<
    | (Partial<TokenBlueprint> & {
        brandName?: string;
      })
    | null
  >(tokenBlueprint);

  const descriptionRef =
    React.useRef<HTMLTextAreaElement | null>(
      null,
    );

  const iconInputRef =
    React.useRef<HTMLInputElement | null>(
      null,
    );

  const canEditIcon = Boolean(
    isEditMode || minted,
  );

  const isIdentityLocked = Boolean(minted);

  React.useEffect(() => {
    let cancelled = false;

    const loadBrands = async () => {
      const brands =
        await loadBrandsForCompany();

      if (!cancelled) {
        setBrandOptions(brands);
      }
    };

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, []);

  React.useEffect(() => {
    const source =
      params.initialTokenBlueprint;

    if (!source) {
      return;
    }

    initialRef.current = source;

    if (isEditMode) {
      return;
    }

    setId(
      pickString(source.id),
    );

    setName(
      pickString(source.name),
    );

    setSymbol(
      pickString(source.symbol),
    );

    setBrandId(
      pickString(source.brandId),
    );

    setBrandName(
      pickBrandName(source),
    );

    setDescription(
      pickString(source.description),
    );

    setMinted(
      typeof source.minted === "boolean"
        ? source.minted
        : false,
    );

    setBurnAt(
      params.initialBurnAt ?? "",
    );

    setSelectedIconFile(null);

    if (localPreviewUrl) {
      URL.revokeObjectURL(
        localPreviewUrl,
      );

      setLocalPreviewUrl("");
    }
  }, [
    params.initialTokenBlueprint,
    params.initialBurnAt,
    isEditMode,
    localPreviewUrl,
    pickBrandName,
    pickString,
  ]);

  React.useEffect(() => {
    if (isEditMode) {
      return;
    }

    setRemoteIconUrl(
      params.initialIconUrl ?? "",
    );
  }, [
    params.initialIconUrl,
    isEditMode,
  ]);

  React.useEffect(() => {
    const element =
      descriptionRef.current;

    if (!element) {
      return;
    }

    element.style.height = "auto";
    element.style.height =
      `${element.scrollHeight}px`;
  }, [description]);

  React.useEffect(() => {
    return () => {
      if (localPreviewUrl) {
        URL.revokeObjectURL(
          localPreviewUrl,
        );
      }
    };
  }, [localPreviewUrl]);

  const requestPickIconFile =
    React.useCallback(() => {
      if (!canEditIcon) {
        return;
      }

      iconInputRef.current?.click();
    }, [canEditIcon]);

  const onIconInputChange =
    React.useCallback(
      (
        event: React.ChangeEvent<HTMLInputElement>,
      ) => {
        if (!canEditIcon) {
          event.target.value = "";
          return;
        }

        const file =
          event.target.files?.[0] ?? null;

        event.target.value = "";

        if (!file) {
          return;
        }

        if (
          !file.type
            ?.toLowerCase()
            .startsWith("image/")
        ) {
          return;
        }

        setSelectedIconFile(file);

        if (localPreviewUrl) {
          URL.revokeObjectURL(
            localPreviewUrl,
          );
        }

        setLocalPreviewUrl(
          URL.createObjectURL(file),
        );
      },
      [
        canEditIcon,
        localPreviewUrl,
      ],
    );

  const shownIconUrl =
    localPreviewUrl || remoteIconUrl;

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
  };

  const handlers: TokenBlueprintCardHandlers = {
    onChangeName: (
      value: string,
    ) => {
      if (isIdentityLocked) {
        return;
      }

      setName(value);
    },

    onChangeSymbol: (
      value: string,
    ) => {
      if (isIdentityLocked) {
        return;
      }

      setSymbol(
        value.toUpperCase(),
      );
    },

    onChangeBrand: (
      nextBrandId: string,
      nextBrandName: string,
    ) => {
      if (isIdentityLocked) {
        return;
      }

      setBrandId(nextBrandId);
      setBrandName(nextBrandName);
    },

    onChangeDescription: (
      value: string,
    ) => {
      setDescription(value);
    },

    iconInputRef,
    descriptionRef,
    onRequestPickIconFile:
      requestPickIconFile,
    onIconInputChange,

    onClearLocalIconFile: () => {
      setSelectedIconFile(null);

      if (localPreviewUrl) {
        URL.revokeObjectURL(
          localPreviewUrl,
        );

        setLocalPreviewUrl("");
      }
    },

    onToggleEditMode: () => {
      setIsEditMode(
        (current) => !current,
      );
    },

    setEditMode: (
      edit: boolean,
    ) => {
      setIsEditMode(edit);
    },

    reset: () => {
      const source =
        initialRef.current;

      if (!source) {
        return;
      }

      setId(
        pickString(source.id),
      );

      setName(
        pickString(source.name),
      );

      setSymbol(
        pickString(source.symbol),
      );

      setBrandId(
        pickString(source.brandId),
      );

      setBrandName(
        pickBrandName(source),
      );

      setDescription(
        pickString(source.description),
      );

      setMinted(
        typeof source.minted === "boolean"
          ? source.minted
          : false,
      );

      setBurnAt(
        params.initialBurnAt ?? "",
      );

      setRemoteIconUrl(
        params.initialIconUrl ?? "",
      );

      setSelectedIconFile(null);

      if (localPreviewUrl) {
        URL.revokeObjectURL(
          localPreviewUrl,
        );

        setLocalPreviewUrl("");
      }
    },
  };

  return {
    vm,
    handlers,
    selectedIconFile,
    burnAt,
    canEditIcon,
  };
}