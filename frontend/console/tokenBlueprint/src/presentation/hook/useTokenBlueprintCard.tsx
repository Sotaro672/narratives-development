// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintCard.tsx
import * as React from "react";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

import type {
  TokenBlueprintCardViewModel,
  TokenBlueprintCardHandlers,
} from "../components/tokenBlueprintCard";

// Service（アプリケーションロジック）
import {
  loadBrandsForCompany,
  resolveBrandName,
} from "../../application/tokenBlueprintCreateService";

/**
 * TokenBlueprintCard 用のロジックフック
 * - UI 状態管理 +（★移譲）アイコンアップロードの UI ロジック（file input / preview / click）
 *
 * ★仕様:
 * - minted は boolean（notYet/minted は使わない）
 * - minted:true でも「トークンアイコンは編集できる」(=アイコンだけアップロード可能)
 * - minted:true の場合、トークン名 / シンボル / ブランドは変更不可
 * - API スキーマは name / brandName を正とする
 */
export function useTokenBlueprintCard(params: {
  initialTokenBlueprint?: Partial<TokenBlueprint> & {
    brandName?: string;
  };
  initialBurnAt?: string;
  initialIconUrl?: string;
  initialEditMode?: boolean;

  // ★ 将来：実アップロードを hook 外から注入したい場合に使う（任意）
  // onUploadIcon?: (file: File, tokenBlueprintId: string) => Promise<void> | void;
}) {
  const tb = params.initialTokenBlueprint ?? {};

  const pickBrandName = React.useCallback((src: any): string => {
    return String(src?.brandName ?? "").trim();
  }, []);

  const pickString = React.useCallback((v: any): string => String(v ?? "").trim(), []);

  // -------------------------
  // Local UI states
  // -------------------------
  const [id, setId] = React.useState(pickString((tb as any).id));
  const [name, setName] = React.useState(pickString((tb as any).name));
  const [symbol, setSymbol] = React.useState(pickString((tb as any).symbol));

  const [brandId, setBrandId] = React.useState(pickString((tb as any).brandId));
  const [brandName, setBrandName] = React.useState(pickBrandName(tb as any));

  const [description, setDescription] = React.useState(pickString((tb as any).description));
  const [burnAt, setBurnAt] = React.useState(params.initialBurnAt ?? "");

  // ★ minted は boolean（未設定は false 扱い）
  const [minted, setMinted] = React.useState<boolean>(
    typeof (tb as any).minted === "boolean" ? (tb as any).minted : false,
  );

  // ★ backend 反映済み iconUrl（初期値）
  const [remoteIconUrl, setRemoteIconUrl] = React.useState(params.initialIconUrl ?? "");

  // ★ ローカルプレビュー（アップロード前に表示したい場合）
  const [localPreviewUrl, setLocalPreviewUrl] = React.useState<string>("");

  // ★ 選択したアイコンファイル（service に渡すために保持）
  const [selectedIconFile, setSelectedIconFile] = React.useState<File | null>(null);

  // ⭐ 編集モード切り替え可能に変更
  const [isEditMode, setIsEditMode] = React.useState(params.initialEditMode ?? false);

  const [brandOptions, setBrandOptions] = React.useState<{ id: string; name: string }[]>([]);

  // リセット用に「バックエンドからもらった元データ」を保持
  const initialRef = React.useRef<
    (Partial<TokenBlueprint> & { brandName?: string }) | null
  >(tb);

  // ★ UI refs（component からはスタイルだけにするため、ここで管理）
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);
  const iconInputRef = React.useRef<HTMLInputElement | null>(null);

  // ★ アイコンは minted:true でも編集可能（編集モード OR minted）
  const canEditIcon = Boolean(isEditMode || minted);

  // ★ minted:true の場合は、トークン名 / シンボル / ブランドをロック
  const isIdentityLocked = Boolean(minted);

  // -------------------------
  // Brand 一覧読み込み（Service に委譲）
  // -------------------------
  React.useEffect(() => {
    let cancelled = false;

    loadBrandsForCompany().then((brands) => {
      if (!cancelled) setBrandOptions(brands);
    });

    return () => {
      cancelled = true;
    };
  }, []);

  // backend から initialTokenBlueprint が更新されたら（= 詳細取得完了したら）state に反映
  React.useEffect(() => {
    const src = params.initialTokenBlueprint as any;
    if (!src) return;

    // リセット用に保持
    initialRef.current = src;

    // 編集中はユーザー入力を壊さない
    if (isEditMode) return;

    setId(src.id ?? "");
    setName(src.name ?? "");
    setSymbol(src.symbol ?? "");
    setBrandId(src.brandId ?? "");
    setBrandName(pickBrandName(src));
    setDescription(src.description ?? "");

    // ★ minted(boolean) を反映（未設定は false）
    setMinted(typeof src.minted === "boolean" ? src.minted : false);

    // burnAt は呼び出し元初期値を維持
    setBurnAt(params.initialBurnAt ?? "");

    // ★ 詳細取得で状態が置き換わったタイミングでは「未アップロードの選択ファイル」は消して安全側に倒す
    setSelectedIconFile(null);
    if (localPreviewUrl) {
      try {
        URL.revokeObjectURL(localPreviewUrl);
      } catch {
        // ignore
      }
      setLocalPreviewUrl("");
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.initialTokenBlueprint, params.initialBurnAt, isEditMode, pickBrandName]);

  // ★ 初期 iconUrl が更新された場合（詳細取得後など）
  React.useEffect(() => {
    if (isEditMode) return;

    const next = params.initialIconUrl ?? "";
    setRemoteIconUrl(next);
    // ローカルプレビューは「ユーザーが選択した時だけ」なのでここでは触らない
  }, [params.initialIconUrl, isEditMode]);

  // brandId しか無い場合、brandName を backend から解決
  React.useEffect(() => {
    let cancelled = false;

    if (!brandId || brandName) return;

    resolveBrandName(brandId).then((name) => {
      if (!cancelled && name) setBrandName(name);
    });

    return () => {
      cancelled = true;
    };
  }, [brandId, brandName]);

  // ★ textarea auto-resize（スタイル/見た目の調整なので hook 側に移譲）
  React.useEffect(() => {
    if (!descriptionRef.current) return;
    descriptionRef.current.style.height = "auto";
    descriptionRef.current.style.height = `${descriptionRef.current.scrollHeight}px`;
  }, [description]);

  // ★ ローカルプレビューのメモリ解放（unmount）
  React.useEffect(() => {
    return () => {
      if (localPreviewUrl) {
        try {
          URL.revokeObjectURL(localPreviewUrl);
        } catch {
          // ignore
        }
      }
    };
  }, [localPreviewUrl]);

  // -------------------------
  // Icon upload UI logic
  // -------------------------
  const requestPickIconFile = React.useCallback(() => {
    // ★ minted:true でも開ける
    if (!canEditIcon) return;
    iconInputRef.current?.click();
  }, [canEditIcon]);

  const onIconInputChange = React.useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      // ★ minted:true でも許可
      if (!canEditIcon) {
        e.target.value = "";
        return;
      }

      const file = e.target.files?.[0] ?? null;
      e.target.value = "";

      if (!file) return;

      if (!file.type?.startsWith("image/")) {
        return;
      }

      setSelectedIconFile(file);

      try {
        if (localPreviewUrl) URL.revokeObjectURL(localPreviewUrl);
        setLocalPreviewUrl(URL.createObjectURL(file));
      } catch {
        // ignore
      }
    },
    [canEditIcon, localPreviewUrl],
  );

  const shownIconUrl = localPreviewUrl || remoteIconUrl;

  // -------------------------
  // ViewModel
  // -------------------------
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

  // -------------------------
  // Handlers（UI のみ）
  // -------------------------
  const handlers: TokenBlueprintCardHandlers = {
    onChangeName: (v) => {
      if (isIdentityLocked) return;
      setName(v);
    },

    onChangeSymbol: (v) => {
      if (isIdentityLocked) return;
      setSymbol(v.toUpperCase());
    },

    onChangeBrand: (id, name) => {
      if (isIdentityLocked) return;
      setBrandId(id);
      setBrandName(name);
    },

    onChangeDescription: (v) => setDescription(v),

    iconInputRef,
    descriptionRef,
    onRequestPickIconFile: requestPickIconFile,
    onIconInputChange,

    onClearLocalIconFile: () => {
      setSelectedIconFile(null);
      if (localPreviewUrl) {
        try {
          URL.revokeObjectURL(localPreviewUrl);
        } catch {
          // ignore
        }
        setLocalPreviewUrl("");
      }
    },

    onPreview: () => {
      alert("プレビュー画面を開きます（モック）");
    },

    onToggleEditMode: () => {
      setIsEditMode((prev) => !prev);
    },

    setEditMode: (edit: boolean) => {
      setIsEditMode(edit);
    },

    reset: () => {
      const src = initialRef.current as any;
      if (!src) return;

      setId(src.id ?? "");
      setName(src.name ?? "");
      setSymbol(src.symbol ?? "");
      setBrandId(src.brandId ?? "");
      setBrandName(pickBrandName(src));
      setDescription(src.description ?? "");

      setMinted(typeof src.minted === "boolean" ? src.minted : false);

      setBurnAt(params.initialBurnAt ?? "");
      setRemoteIconUrl(params.initialIconUrl ?? "");

      setSelectedIconFile(null);
      if (localPreviewUrl) {
        try {
          URL.revokeObjectURL(localPreviewUrl);
        } catch {
          // ignore
        }
        setLocalPreviewUrl("");
      }
    },
  };

  return { vm, handlers, selectedIconFile, burnAt, canEditIcon };
}