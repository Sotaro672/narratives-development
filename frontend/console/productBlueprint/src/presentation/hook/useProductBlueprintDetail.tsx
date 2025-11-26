// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

import type { ProductIDTagType } from "../../../../shell/src/shared/types/productBlueprint";
import {
  brandLabelFromId,
  formatProductBlueprintDate,
  type SizeRow,
  type ModelNumberRow,
} from "../../infrastructure/api/productBlueprintApi";

import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
} from "../../application/productBlueprintDetailService";

import type { Fit, ItemType, WashTagOption } from "../../domain/entity/catalog";

export {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  WASH_TAG_OPTIONS,
} from "../../domain/entity/catalog";
export type { Fit, WashTagOption } from "../../domain/entity/catalog";

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

  colors: string[];
  colorInput: string;
  sizes: SizeRow[];
  modelNumbers: ModelNumberRow[];

  /** color 名 → HEX(RGB) のマップ（例: { "グリーン": "#00ff00" }） */
  colorRgbMap: Record<string, string>;

  getCode: (sizeLabel: string, color: string) => string;

  assignee: string;
  creator: string;
  createdAt: string;

  onBack: () => void;
  onSave: () => void;

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

  onRemoveSize: (id: string) => void;

  onEditAssignee: () => void;
  onClickAssignee: () => void;
  onClickCreatedBy: () => void;
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

  const [colorInput, setColorInput] = React.useState("");
  const [colors, setColors] = React.useState<string[]>([]);
  const [sizes, setSizes] = React.useState<SizeRow[]>([]);
  const [modelNumbers, setModelNumbers] = React.useState<ModelNumberRow[]>([]);

  // Color.Name / color.rgb を HEX(#rrggbb) にして保持
  const [colorRgbMap, setColorRgbMap] = React.useState<Record<string, string>>(
    {},
  );

  const [assignee, setAssignee] = React.useState("担当者未設定");
  const [creator, setCreator] = React.useState("作成者未設定");
  const [createdAt, setCreatedAt] = React.useState("");

  // ---------------------------------
  // service → 詳細データ + variations を反映
  // ---------------------------------
  React.useEffect(() => {
    if (!blueprintId) return;

    (async () => {
      try {
        const detail = await getProductBlueprintDetail(blueprintId);

        console.log("[useProductBlueprintDetail] mapped detail:", detail);

        const brandNameFromService = (detail as any).brandName as
          | string
          | undefined;
        const assigneeNameFromService = (detail as any).assigneeName as
          | string
          | undefined;
        const createdByNameFromService = (detail as any).createdByName as
          | string
          | undefined;

        const productBlueprintId = detail.id ?? blueprintId;

        setPageTitle(detail.productName ?? productBlueprintId);
        setProductName(detail.productName ?? "");

        setBrand(
          brandNameFromService ?? brandLabelFromId(detail.brandId),
        );

        setItemType((detail.itemType as ItemType) ?? "");
        setFit((detail.fit as Fit) ?? ("" as Fit));

        setMaterials(detail.material ?? "");
        setWeight(detail.weight ?? 0);
        setWashTags(detail.qualityAssurance ?? []);

        const tagType =
          (detail.productIdTag?.type as ProductIDTagType | undefined) ?? "";
        setProductIdTagType(tagType);

        // --------------------------------------------------
        // ModelVariation 取得
        // --------------------------------------------------
        try {
          const variations =
            await listModelVariationsByProductBlueprintId(
              productBlueprintId,
            );

          console.log(
            "[useProductBlueprintDetail] model variations:",
            variations,
          );

          // colors: variation.color.name のユニーク集合
          const uniqueColors = Array.from(
            new Set(
              variations
                .map((v) => v.color?.name?.trim())
                .filter((c): c is string => !!c),
            ),
          );
          setColors(uniqueColors);

          // サイズ: variation.size のユニーク集合
          const uniqueSizes = Array.from(
            new Set(
              variations
                .map((v) => v.size?.trim())
                .filter((s): s is string => !!s),
            ),
          );

          const sizeRows: SizeRow[] = uniqueSizes.map((label, index) => ({
            id: String(index + 1),
            sizeLabel: label,
          })) as SizeRow[];
          setSizes(sizeRows);

          // modelNumbers: size × color ごとのコード
          const modelNumberRows: ModelNumberRow[] = variations.map((v) => ({
            size: v.size,
            color: v.color?.name ?? "",
            code: v.modelNumber,
          }));
          setModelNumbers(modelNumberRows);

          // colorRgbMap: Color.rgb(number) → HEX(#rrggbb)
          const nextColorRgbMap: Record<string, string> = {};
          for (const v of variations) {
            const name = v.color?.name?.trim();
            const rgb = v.color?.rgb;
            if (!name || typeof rgb !== "number") continue;

            const hex =
              "#" +
              rgb
                .toString(16)
                .padStart(6, "0")
                .toLowerCase();
            nextColorRgbMap[name] = hex;
          }
          setColorRgbMap(nextColorRgbMap);
        } catch (e) {
          console.error(
            "[useProductBlueprintDetail] listModelVariationsByProductBlueprintId failed:",
            e,
          );
          setColors([]);
          setSizes([]);
          setModelNumbers([]);
          setColorRgbMap({});
        }

        // assignee
        setAssignee(
          assigneeNameFromService ??
            detail.assigneeId ??
            "担当者未設定",
        );

        // creator
        setCreator(
          createdByNameFromService ??
            detail.createdBy ??
            "作成者未設定",
        );

        setCreatedAt(formatProductBlueprintDate(detail.createdAt) || "");
      } catch (e) {
        console.error("[useProductBlueprintDetail] fetch failed:", e);
      }
    })();
  }, [blueprintId]);

  // ---------------------------------
  // ModelNumberCard 用
  // ---------------------------------
  const getCode = React.useCallback(
    (sizeLabel: string, color: string): string => {
      const row = modelNumbers.find(
        (m) => m.size === sizeLabel && m.color === color,
      );
      return row?.code ?? "";
    },
    [modelNumbers],
  );

  // ---------------------------------
  // Handlers
  // ---------------------------------
  const onSave = React.useCallback(() => {
    alert("保存しました（ダミー）");
  }, []);

  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

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
    setAssignee("新担当者");
  }, []);

  const onClickAssignee = React.useCallback(() => {
    console.log("assignee clicked:", assignee);
  }, [assignee]);

  const onClickCreatedBy = React.useCallback(() => {
    console.log("createdBy clicked:", creator);
  }, [creator]);

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

    colors,
    colorInput,
    sizes,
    modelNumbers,

    colorRgbMap,

    getCode,

    assignee,
    creator,
    createdAt,

    onBack,
    onSave,

    onChangeProductName: setProductName,
    onChangeItemType: (v: ItemType) => setItemType(v),
    onChangeFit: setFit,
    onChangeMaterials: setMaterials,
    onChangeWeight: setWeight,
    onChangeWashTags: setWashTags,
    onChangeProductIdTag: (v: string) =>
      setProductIdTagType(v as ProductIDTagType),

    onChangeColorInput: setColorInput,
    onAddColor,
    onRemoveColor,

    onRemoveSize,

    onEditAssignee,
    onClickAssignee,
    onClickCreatedBy,
  };
}
