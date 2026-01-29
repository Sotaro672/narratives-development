// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";

import type {
  SizeRow,
  ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  updateProductBlueprint,
  softDeleteProductBlueprint,
} from "../../application/productBlueprintDetailService";

import type { Fit, ItemType } from "../../domain/entity/catalog";

import { formatDateTimeYYYYMMDDHHmm } from "../util/dateFormat";
import { mapVariationsToUiState } from "../util/variationMapper";
import { useBrandOptions, type BrandOption } from "./useBrandOptions";
import {
  useVariationsEditor,
  type VariationsUiState,
} from "./useVariationsEditor";

export {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
export type { Fit, WashTagOption } from "../../domain/entity/catalog";

// ------------------------------
// ✅ displayOrder で variations を並べ替えるユーティリティ
// ------------------------------
type ModelRefLike = { modelId?: string; displayOrder?: number };

/**
 * modelRefs(displayOrder) に従い variations を並べ替える。
 * - refs に存在する id は displayOrder 昇順で先頭に並べる
 * - refs に無い variations は “元の順” のまま末尾に回す（情報欠損時の保険）
 */
function orderVariationsByModelRefs(
  variations: any[],
  modelRefs: ModelRefLike[] | undefined,
): any[] {
  if (!Array.isArray(variations) || variations.length === 0) return [];
  if (!Array.isArray(modelRefs) || modelRefs.length === 0) return variations;

  const byId = new Map<string, any>();
  for (const v of variations) {
    const id = typeof v?.id === "string" ? v.id.trim() : "";
    if (!id) continue;
    // 同一 id は先勝ち（後勝ちにしたいならここを上書きに変更）
    if (!byId.has(id)) byId.set(id, v);
  }

  const sortedRefs = [...modelRefs]
    .map((r) => ({
      id: typeof r?.modelId === "string" ? r.modelId.trim() : "",
      order: typeof r?.displayOrder === "number" ? r.displayOrder : Number.NaN,
    }))
    .filter((x) => x.id && Number.isFinite(x.order))
    .sort((a, b) => a.order - b.order);

  const used = new Set<string>();
  const ordered: any[] = [];

  for (const ref of sortedRefs) {
    const v = byId.get(ref.id);
    if (!v) continue;
    if (used.has(ref.id)) continue;
    used.add(ref.id);
    ordered.push(v);
  }

  // refs に無い variations は元順を維持して末尾へ
  for (const v of variations) {
    const id = typeof v?.id === "string" ? v.id.trim() : "";
    if (!id) continue;
    if (used.has(id)) continue;
    used.add(id);
    ordered.push(v);
  }

  return ordered;
}

export interface UseProductBlueprintDetailResult {
  pageTitle: string;

  productName: string;
  brand: string;
  itemType: ItemType | "";
  fit: Fit;
  materials: string;
  weight: number;
  washTags: string[];
  productIdTag: ProductIDTagType | "";

  /** ブランド編集用 */
  brandId: string;
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  /** color 名 → rgb hex (#rrggbb) */
  colorRgbMap: Record<string, string>;

  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;

  /** 管理情報 */
  creator: string;
  createdAt: string;
  updater: string;
  updatedAt: string;

  /** ✅ 印刷済みかどうか（printed:true の場合は編集ボタンを非表示にする） */
  printed: boolean;

  onBack: () => void;
  onSave: () => void;
  /** 論理削除（削除ボタン用） */
  onDelete: () => void;

  onChangeProductName: (v: string) => void;
  onChangeItemType: (v: ItemType) => void;
  onChangeFit: (v: Fit) => void;
  onChangeMaterials: (v: string) => void;
  onChangeWeight: (v: number) => void;
  onChangeWashTags: (v: string[]) => void;
  onChangeProductIdTag: (v: string) => void;

  onChangeColorInput: (v: string) => void;
  onAddColor: () => void;
  onRemoveColor: (name: string) => void;
  /** カラーの RGB(hex) 更新用 */
  onChangeColorRgb: (name: string, hex: string) => void;

  /** サイズ削除 */
  onRemoveSize: (id: string) => void;
  /** サイズ追加（SizeVariationCard の「サイズを追加」ボタン用） */
  onAddSize: () => void;
  /** サイズ 1 行分の変更（SizeVariationCard の各セル編集用） */
  onChangeSize: (id: string, patch: Partial<Omit<SizeRow, "id">>) => void;

  /** モデルナンバー変更（ModelNumberCard のセル編集用） */
  onChangeModelNumber: (
    sizeLabel: string,
    color: string,
    nextCode: string,
  ) => void;

  onClickAssignee: () => void;
}

export function useProductBlueprintDetail(): UseProductBlueprintDetailResult {
  const navigate = useNavigate();
  const { blueprintId } = useParams<{ blueprintId: string }>();

  const [pageTitle, setPageTitle] = React.useState<string>("");

  const [productName, setProductName] = React.useState<string>("");
  const [brand, setBrand] = React.useState<string>("");

  const [itemType, setItemType] = React.useState<ItemType | "">("");
  const [fit, setFit] = React.useState<Fit>("" as Fit);

  const [materials, setMaterials] = React.useState<string>("");
  const [weight, setWeight] = React.useState<number>(0);
  const [washTags, setWashTags] = React.useState<string[]>([]);

  const [productIdTagType, setProductIdTagType] =
    React.useState<ProductIDTagType | "">("");

  const [assignee, setAssignee] = React.useState("担当者未設定");

  // ★ 管理情報（updater/updatedAt は「未設定文字列」で埋めず、空文字で管理する）
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");
  const [updater, setUpdater] = React.useState("");
  const [updatedAt, setUpdatedAt] = React.useState("");

  // ✅ printed:true の場合にヘッダーの編集ボタンを非表示にするため保持
  const [printed, setPrinted] = React.useState<boolean>(false);

  // ★ サーバに渡すための ID を保持
  const [brandId, setBrandId] = React.useState<string>("");
  const [assigneeId, setAssigneeId] = React.useState<string>("");

  // brand resolution inputs（service response 由来）
  const [companyId, setCompanyId] = React.useState<string>("");
  const [brandNameFromService, setBrandNameFromService] =
    React.useState<string>("");

  // --------------------------------------------------
  // variations editor (colors/sizes/modelNumbers/rgbMap) を専用 hook に委譲
  // --------------------------------------------------
  const {
    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,
    getCode,
    setFromUiState,
    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,
    onRemoveSize,
    onAddSize,
    onChangeSize,
    onChangeModelNumber,
  } = useVariationsEditor();

  // --------------------------------------------------
  // ブランド一覧取得 + brandId -> name 解決は専用 hook に委譲
  // --------------------------------------------------
  const {
    brandOptions,
    brandLoading,
    brandError,
    resolvedBrandName,
    getBrandNameById,
  } = useBrandOptions({
    companyId,
    brandId,
    brandNameFromService,
  });

  // ---------------------------------
  // service → 詳細データ + variations を反映
  // ---------------------------------
  React.useEffect(() => {
    if (!blueprintId) return;

    (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        const brandNameSvc = String((detail as any).brandName ?? "").trim();
        const assigneeNameFromService = (detail as any).assigneeName as
          | string
          | undefined;

        const createdByNameFromService = (detail as any).createdByName as
          | string
          | undefined;

        // ★ 追加: 最終更新者（表示名）
        const updatedByNameFromService = (detail as any).updatedByName as
          | string
          | undefined;

        const productBlueprintIdResolved = detail.id ?? blueprintId;
        const itemTypeFromDetail = detail.itemType as ItemType;

        setPageTitle(detail.productName ?? productBlueprintIdResolved);
        setProductName(detail.productName ?? "");

        // ✅ printed（dto を正: camelCase の printed を読む）
        setPrinted(Boolean((detail as any).printed));

        // brandId / assigneeId を state に保持
        setBrandId(detail.brandId ?? "");
        setAssigneeId(detail.assigneeId ?? "");

        // brand hook inputs
        setCompanyId(detail.companyId ?? "");
        setBrandNameFromService(brandNameSvc);

        setItemType(itemTypeFromDetail ?? "");
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const tagType =
          (detail.productIdTag?.type as ProductIDTagType | undefined) ?? "";
        setProductIdTagType(tagType);

        // ✅ modelRefs（displayOrder のソース）
        const modelRefs = (detail as any).modelRefs as ModelRefLike[] | undefined;

        // --------------------------------------------------
        // ModelVariation 取得 → displayOrder で整列 → UI state 変換
        // --------------------------------------------------
        try {
          const variations = await listModelVariationsByProductBlueprintId(
            productBlueprintIdResolved,
          );

          // ✅ 最重要: displayOrder で variations を並べ替える
          const ordered = orderVariationsByModelRefs(
            variations as any[],
            modelRefs,
          );

          const uiState = mapVariationsToUiState({
            varsAny: ordered as any[],
            itemType: itemTypeFromDetail,
          });

          // editor へ反映（colorInput はクリアされる）
          setFromUiState(uiState as VariationsUiState);
        } catch (e) {
          console.error(
            "[useProductBlueprintDetail] listModelVariationsByProductBlueprintId failed:",
            e,
          );
          setFromUiState({
            colors: [],
            sizes: [],
            modelNumbers: [],
            colorRgbMap: {},
          });
        }

        // assignee
        setAssignee(
          assigneeNameFromService ?? detail.assigneeId ?? "担当者未設定",
        );

        // creator
        setCreator(
          createdByNameFromService ?? detail.createdBy ?? "作成者未設定",
        );

        // createdAt は HH:mm まで表示
        setCreatedAt(
          formatDateTimeYYYYMMDDHHmm((detail as any).createdAt) || "",
        );

        // ★ updater/updatedAt は「両方揃っている時だけ」セットする
        const updatedByRaw =
          (updatedByNameFromService ?? (detail as any).updatedBy ?? "") as any;
        const updaterName = String(updatedByRaw ?? "").trim();
        const updatedAtDisp =
          formatDateTimeYYYYMMDDHHmm((detail as any).updatedAt) || "";

        if (!updaterName || !updatedAtDisp) {
          setUpdater("");
          setUpdatedAt("");
        } else {
          setUpdater(updaterName);
          setUpdatedAt(updatedAtDisp);
        }
      } catch (e) {
        console.error("[useProductBlueprintDetail] fetch failed:", e);
      }
    })();
  }, [blueprintId, setFromUiState]);

  // brand: hook が解決した name を表示に反映
  React.useEffect(() => {
    setBrand(resolvedBrandName ?? "");
  }, [resolvedBrandName]);

  // ---------------------------------
  // Handlers: 保存
  // ---------------------------------
  const onSave = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計ID が不明です");
      return;
    }

    if (!itemType) {
      alert("アイテム種別を選択してください");
      return;
    }

    const hasEmptyModelNumber = sizes.some((s) => {
      const sizeLabel = (s.sizeLabel ?? "").trim();
      if (!sizeLabel) return false;

      return colors.some((c) => {
        const color = (c ?? "").trim();
        if (!color) return false;

        const code = getCode(sizeLabel, color);
        return !code || !code.trim();
      });
    });

    if (hasEmptyModelNumber) {
      alert("モデルナンバーが空欄です");
      return;
    }

    (async () => {
      try {
        console.log("[useProductBlueprintDetail] onSave payload snapshot", {
          blueprintId,
          productName,
          itemType,
          fit,
          materials,
          weight,
          washTags,
          productIdTagType,
          sizes,
          modelNumbers,
          colorRgbMap,
          brandId,
          assigneeId,
          printed,
        });

        await updateProductBlueprint({
          id: blueprintId,
          productName,
          itemType: itemType as ItemType,
          fit,
          material: materials,
          weight,
          qualityAssurance: washTags,
          productIdTag: productIdTagType
            ? { type: productIdTagType }
            : undefined,
          sizes,
          modelNumbers,
          colorRgbMap,
          brandId,
          assigneeId,
        } as any);

        alert("保存しました");
      } catch (e) {
        console.error("[useProductBlueprintDetail] update failed:", e);
        alert("保存に失敗しました");
      }
    })();
  }, [
    blueprintId,
    productName,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTagType,
    sizes,
    modelNumbers,
    colorRgbMap,
    brandId,
    assigneeId,
    colors,
    getCode,
    printed,
  ]);

  // ---------------------------------
  // Handlers: 論理削除
  // ---------------------------------
  const onDelete = React.useCallback(() => {
    if (!blueprintId) {
      alert("商品設計ID が不明です");
      return;
    }

    const ok = window.confirm(
      "この商品設計を削除しますか？\n関連するモデルバリエーションも含めて論理削除されます。",
    );
    if (!ok) return;

    (async () => {
      try {
        await softDeleteProductBlueprint(blueprintId);
        alert("削除しました");
        navigate("/productBlueprint");
      } catch (e) {
        console.error("[useProductBlueprintDetail] delete failed:", e);
        alert("削除に失敗しました");
      }
    })();
  }, [blueprintId, navigate]);

  const onBack = React.useCallback(() => {
    navigate("/productBlueprint");
  }, [navigate]);

  const onClickAssignee = React.useCallback(() => {
    console.log("assignee clicked:", assignee);
  }, [assignee]);

  const onChangeBrandId = React.useCallback(
    (id: string) => {
      const nextId = String(id ?? "").trim();
      setBrandId(nextId);

      // 表示名も即時更新（options が揃っていれば）
      const nextName = getBrandNameById(nextId);
      if (nextName) {
        setBrand(nextName);
      } else {
        setBrand(brandNameFromService || "");
      }
    },
    [getBrandNameById, brandNameFromService],
  );

  return {
    pageTitle,

    productName,
    brand,
    itemType,
    fit,
    materials,
    weight,
    washTags,
    productIdTag: productIdTagType || "",

    brandId,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,

    colors,
    colorInput,
    sizes,
    modelNumbers,
    colorRgbMap,

    getCode,

    assignee,

    creator,
    createdAt,
    updater,
    updatedAt,

    printed,

    onBack,
    onSave,
    onDelete,

    onChangeProductName: setProductName,
    onChangeItemType: (v: ItemType) => setItemType(v),
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,
    onChangeProductIdTag: (v: string) =>
      setProductIdTagType(v as ProductIDTagType),

    onChangeColorInput,
    onAddColor,
    onRemoveColor,
    onChangeColorRgb,

    onRemoveSize,
    onAddSize,
    onChangeSize,

    onChangeModelNumber,

    onClickAssignee,
  };
}
