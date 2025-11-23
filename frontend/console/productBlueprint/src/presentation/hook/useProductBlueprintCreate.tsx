// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type {
  ItemType,
  ProductIDTagType,
} from "../../domain/entity/productBlueprint";

// Brand (domain)
import type { Brand } from "../../../../brand/src/domain/entity/brand";
// ★ companyId フィルタ付きの安全なクエリを利用
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

// Size / ModelNumber の型だけ借りる
import type { SizeRow } from "../../../../model/src/presentation/components/SizeVariationCard";
import type { ModelNumber } from "../../../../model/src/presentation/components/ModelNumberCard";

// Auth / currentMember
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// Fit は詳細画面と同じユニオン型
export type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

// -------------------------------
// Hook が外に公開する ViewModel
// -------------------------------
export interface UseProductBlueprintCreateResult {
  // Meta
  title: string;

  // ブランド関連
  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  // 商品設計フィールド
  productName: string;
  itemType: ItemType;
  fit: Fit;
  material: string;
  weight: number;
  qualityAssurance: string[];
  productIdTagType: ProductIDTagType;

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumber[];

  assigneeId: string;
  createdBy: string;
  createdAt: string;

  // 画面全体アクション
  onCreate: () => void;
  onBack: () => void;

  // 入力変更ハンドラ（ページから渡して使う）
  onChangeProductName: (v: string) => void;
  onChangeItemType: (v: ItemType) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterial: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeQualityAssurance: (v: string[]) => void;
  onChangeProductIdTagType: (v: ProductIDTagType) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  onRemoveSize: (id: string) => void;

  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onClickCreatedBy: () => void;
}

/**
 * 商品設計作成画面用のロジック・状態をまとめたカスタムフック
 * - brands は companyId でフィルタされた安全なクエリ(fetchAllBrandsForCompany)のみ利用
 * - 担当者(assignee) は currentMember を元に自動設定
 */
export function useProductBlueprintCreate(): UseProductBlueprintCreateResult {
  const navigate = useNavigate();

  // Auth / currentMember から companyId を取得
  const { currentMember, user } = useAuth();
  const effectiveCompanyId = React.useMemo(
    () => (currentMember?.companyId ?? user?.companyId ?? "").trim(),
    [currentMember?.companyId, user?.companyId],
  );

  // ───────────────────────
  // ブランド一覧（companyId で必ず絞る）
  // ───────────────────────
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<Brand[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      // companyId がまだ取れていない間は何もしない
      if (!effectiveCompanyId) {
        console.log(
          "[useProductBlueprintCreate] companyId is empty; skip brand fetch.",
        );
        setBrandOptions([]);
        return;
      }

      console.log(
        "[useProductBlueprintCreate] start fetchAllBrandsForCompany",
        { companyId: effectiveCompanyId },
      );

      setBrandLoading(true);
      setBrandError(null);

      try {
        const items = await fetchAllBrandsForCompany(
          effectiveCompanyId,
          true, // isActiveOnly
        );
        if (cancelled) {
          console.log(
            "[useProductBlueprintCreate] brand fetch result ignored (cancelled)",
          );
          return;
        }

        console.log(
          "[useProductBlueprintCreate] fetchAllBrandsForCompany result",
          {
            count: items.length,
            sample: items.slice(0, 3),
          },
        );

        setBrandOptions(items);

        // ★ ここでは brandId を自動選択しない（初期状態は未選択のまま）
      } catch (e) {
        const err = e instanceof Error ? e : new Error(String(e));
        if (!cancelled) {
          setBrandError(err);
        }
        console.error(
          "[useProductBlueprintCreate] failed to fetch brands for company:",
          err,
        );
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, [effectiveCompanyId]);

  // 選択中ブランド名
  const brandName = React.useMemo(() => {
    const found = brandOptions.find((b) => b.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  // ───────────────────────
  // 商品設計フィールド
  // ───────────────────────
  const [productName, setProductName] = React.useState("");
  const [itemType, setItemType] = React.useState<ItemType>("tops");

  // ★ フィットは自動で既定値にせず、空からスタート
  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [material, setMaterial] = React.useState("");
  const [weight, setWeight] = React.useState<number>(0);
  const [qualityAssurance, setQualityAssurance] = React.useState<string[]>([]);

  // ★ 商品IDタグも自動選択せず、空からスタート
  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType>("" as ProductIDTagType);

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers] = React.useState<ModelNumber[]>([]);

  // ───────────────────────
  // 管理情報
  //   - assigneeId は currentMember を元に自動設定
  //   - createdBy / createdAt は現状未使用だがフィールドとして保持
  // ───────────────────────
  const [assigneeId, setAssigneeId] = React.useState("");
  const [createdBy] = React.useState("");
  const [createdAt] = React.useState("");

  // currentMember から担当者名を自動設定
  React.useEffect(() => {
    if (!currentMember) return;

    // すでに手動で設定されている場合は上書きしない
    if (assigneeId) return;

    const label =
      currentMember.fullName ||
      currentMember.email ||
      currentMember.id;

    setAssigneeId(label);
  }, [currentMember, assigneeId]);

  // ───────────────────────
  // アクション
  // ───────────────────────
  const onCreate = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] onCreate payload snapshot", {
      productName,
      brandId,
      brandName,
      itemType,
      fit,
      material,
      weight,
      qualityAssurance,
      productIdTagType,
      colors,
      sizes,
      modelNumbers,
      assigneeId,
      createdBy,
      createdAt,
      companyId: effectiveCompanyId,
    });

    alert("商品設計を作成しました（ダミー）");
    navigate(-1);
  }, [
    productName,
    brandId,
    brandName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,
    colors,
    sizes,
    modelNumbers,
    assigneeId,
    createdBy,
    createdAt,
    effectiveCompanyId,
    navigate,
  ]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  // カラー追加/削除
  const onAddColor = React.useCallback(() => {
    const v = colorInput.trim();
    if (!v || colors.includes(v)) return;
    setColors((prev) => [...prev, v]);
    setColorInput("");
  }, [colorInput, colors]);

  const onRemoveColor = React.useCallback((name: string) => {
    setColors((prev) => prev.filter((c) => c !== name));
  }, []);

  const onRemoveSize = React.useCallback((id: string) => {
    setSizes((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const onEditAssignee = React.useCallback(() => {
    // 担当者編集時も currentMember を優先して再設定しておく
    if (currentMember) {
      const label =
        currentMember.fullName ||
        currentMember.email ||
        currentMember.id;
      setAssigneeId(label);
    }
  }, [currentMember]);

  const onClickAssignee = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] assigneeId clicked:", assigneeId);
  }, [assigneeId]);

  const onClickCreatedBy = React.useCallback(() => {
    console.log("[useProductBlueprintCreate] createdBy clicked:", createdBy);
  }, [createdBy]);

  // -------------------------------
  // 返却する ViewModel
  // -------------------------------
  return {
    title: "商品設計を作成",

    // brand
    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId: (id: string) => setBrandId(id),

    // fields
    productName,
    itemType,
    fit,
    material,
    weight,
    qualityAssurance,
    productIdTagType,

    colors,
    colorInput,
    sizes,
    modelNumbers,

    assigneeId,
    createdBy,
    createdAt,

    // page actions
    onCreate,
    onBack,

    // field handlers
    onChangeProductName: setProductName,
    onChangeItemType: setItemType,
    onChangeFit: setFit,
    onChangeMaterial: setMaterial,
    onChangeWeight: setWeight,
    onChangeQualityAssurance: setQualityAssurance,
    onChangeProductIdTagType: setProductIdTagType,

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,
    onRemoveSize,

    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,
  };
}
