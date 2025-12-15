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
 */
export function useTokenBlueprintCard(params: {
  initialTokenBlueprint?: Partial<TokenBlueprint> & { brandName?: string };
  initialBurnAt?: string;
  initialIconUrl?: string;
  initialEditMode?: boolean;

  // ★ 将来：実アップロードを hook 外から注入したい場合に使う（任意）
  // onUploadIcon?: (file: File, tokenBlueprintId: string) => Promise<void> | void;
}) {
  const tb = params.initialTokenBlueprint ?? {};

  // -------------------------
  // Local UI states
  // -------------------------
  const [id, setId] = React.useState(tb.id ?? "");
  const [name, setName] = React.useState(tb.name ?? "");
  const [symbol, setSymbol] = React.useState(tb.symbol ?? "");

  const [brandId, setBrandId] = React.useState(tb.brandId ?? "");
  const [brandName, setBrandName] = React.useState((tb as any).brandName ?? "");

  const [description, setDescription] = React.useState(tb.description ?? "");
  const [burnAt, setBurnAt] = React.useState(params.initialBurnAt ?? "");

  // ★ minted は boolean（未設定は false 扱い）
  const [minted, setMinted] = React.useState<boolean>(
    typeof (tb as any).minted === "boolean" ? (tb as any).minted : false,
  );

  // ★ backend 反映済み iconUrl（初期値）
  const [remoteIconUrl, setRemoteIconUrl] = React.useState(
    params.initialIconUrl ?? "",
  );

  // ★ ローカルプレビュー（アップロード前に表示したい場合）
  const [localPreviewUrl, setLocalPreviewUrl] = React.useState<string>("");

  // ★ 選択したアイコンファイル（service に渡すために保持）
  const [selectedIconFile, setSelectedIconFile] = React.useState<File | null>(
    null,
  );

  // ⭐ 編集モード切り替え可能に変更
  const [isEditMode, setIsEditMode] = React.useState(
    params.initialEditMode ?? false,
  );

  const [brandOptions, setBrandOptions] = React.useState<
    { id: string; name: string }[]
  >([]);

  // リセット用に「バックエンドからもらった元データ」を保持
  const initialRef = React.useRef<
    (Partial<TokenBlueprint> & { brandName?: string }) | null
  >(tb);

  // ★ UI refs（component からはスタイルだけにするため、ここで管理）
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);
  const iconInputRef = React.useRef<HTMLInputElement | null>(null);

  // ★ アイコンは minted:true でも編集可能（編集モード OR minted）
  const canEditIcon = Boolean(isEditMode || minted);

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
    const src = params.initialTokenBlueprint;
    if (!src) return;

    // リセット用に保持
    initialRef.current = src;

    // 編集中はユーザー入力を壊さない
    if (isEditMode) return;

    setId(src.id ?? "");
    setName(src.name ?? "");
    setSymbol(src.symbol ?? "");
    setBrandId(src.brandId ?? "");
    setBrandName((src as any).brandName ?? "");
    setDescription(src.description ?? "");

    // ★ minted(boolean) を反映（未設定は false）
    setMinted(
      typeof (src as any).minted === "boolean" ? (src as any).minted : false,
    );

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

    // burnAt は今のところ別初期値を優先
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [params.initialTokenBlueprint, isEditMode]);

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
        // 同じファイルを連続で選べるように value をクリア
        e.target.value = "";
        return;
      }

      const file = e.target.files?.[0] ?? null;

      // 同じファイルを連続で選べるように value をクリア（hook 側で実施）
      e.target.value = "";

      if (!file) return;

      // 画像のみ
      if (!file.type?.startsWith("image/")) {
        return;
      }

      // ★ File 本体を保持（service に渡すため）
      setSelectedIconFile(file);

      // ローカルプレビュー
      try {
        if (localPreviewUrl) URL.revokeObjectURL(localPreviewUrl);
        setLocalPreviewUrl(URL.createObjectURL(file));
      } catch {
        // ignore
      }

      // eslint-disable-next-line no-console
      console.log("[useTokenBlueprintCard] icon selected", {
        tokenBlueprintId: id,
        fileName: file.name,
        size: file.size,
        type: file.type,
        minted,
        isEditMode,
        storedToState: true,
      });
    },
    [canEditIcon, id, isEditMode, localPreviewUrl, minted],
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

    minted, // ★ boolean
    isEditMode,
    brandOptions,
  };

  // -------------------------
  // Handlers（UI のみ）
  // -------------------------
  const handlers: TokenBlueprintCardHandlers = {
    onChangeName: (v) => setName(v),
    onChangeSymbol: (v) => setSymbol(v.toUpperCase()),

    onChangeBrand: (id, name) => {
      setBrandId(id);
      setBrandName(name);
    },

    onChangeDescription: (v) => setDescription(v),

    // ★ component 側は style のみ：file picker / onChange は hook 側が持つ
    iconInputRef,
    descriptionRef,
    onRequestPickIconFile: requestPickIconFile,
    onIconInputChange,

    onPreview: () => {
      alert("プレビュー画面を開きます（モック）");
    },

    // ⭐ 既存：トグル
    onToggleEditMode: () => {
      setIsEditMode((prev) => !prev);
    },

    // ⭐ 追加：外側から直接モード指定
    setEditMode: (edit: boolean) => {
      setIsEditMode(edit);
    },

    // ⭐ 追加：元データにリセット
    reset: () => {
      const src = initialRef.current;
      if (!src) return;

      setId(src.id ?? "");
      setName(src.name ?? "");
      setSymbol(src.symbol ?? "");
      setBrandId(src.brandId ?? "");
      setBrandName((src as any).brandName ?? "");
      setDescription(src.description ?? "");

      // ★ minted(boolean) を元に戻す
      setMinted(
        typeof (src as any).minted === "boolean" ? (src as any).minted : false,
      );

      // burnAt は今のところそのまま

      // remoteIconUrl は initialIconUrl を優先
      setRemoteIconUrl(params.initialIconUrl ?? "");

      // ★ 未アップロードの選択ファイル/プレビューはリセット
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

  return { vm, handlers, selectedIconFile };
}
