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

  /** color 名 → rgb hex (#rrggbb) */
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
        const itemTypeFromDetail = detail.itemType as ItemType;

        setPageTitle(detail.productName ?? productBlueprintId);
        setProductName(detail.productName ?? "");

        setBrand(
          brandNameFromService ?? brandLabelFromId(detail.brandId),
        );

        setItemType(itemTypeFromDetail ?? "");
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

          const varsAny = variations as any[];

          // -------------------------------
          // colors（Color.Name / color.name）
          // -------------------------------
          const uniqueColors = Array.from(
            new Set(
              varsAny
                .map((v) => {
                  const nm =
                    typeof v.color?.name === "string"
                      ? v.color.name
                      : typeof v.Color?.Name === "string"
                        ? v.Color.Name
                        : "";
                  return nm.trim();
                })
                .filter((c: string) => !!c),
            ),
          );
          setColors(uniqueColors);

          // -------------------------------
          // sizes（Size / size）+ measurements を反映
          // -------------------------------
          const uniqueSizes = Array.from(
            new Set(
              varsAny
                .map((v) => {
                  const sz =
                    typeof v.size === "string"
                      ? v.size
                      : typeof v.Size === "string"
                        ? v.Size
                        : "";
                  return sz.trim();
                })
                .filter((s: string) => !!s),
            ),
          );

          const sizeRows: SizeRow[] = uniqueSizes.map((label, index) => {
            // any ベースで組み立ててから SizeRow にキャストする
            const base: any = {
              id: String(index + 1),
              sizeLabel: label,
            };

            // 該当サイズの最初の variation
            const found = varsAny.find((v) => {
              const sz =
                typeof v.size === "string"
                  ? v.size
                  : typeof v.Size === "string"
                    ? v.Size
                    : "";
              return sz.trim() === label;
            });

            const ms: Record<string, number | null> | undefined =
              found?.measurements ?? found?.Measurements;

            if (ms && typeof ms === "object") {
              if (itemTypeFromDetail === "ボトムス") {
                // ボトムス用: Firestore の日本語キー → base.* にマッピング
                base.waist = ms["ウエスト"] ?? undefined;
                base.hip = ms["ヒップ"] ?? undefined;
                base.rise = ms["股上"] ?? undefined;
                base.inseam = ms["股下"] ?? undefined;
                base.thighWidth = ms["わたり幅"] ?? undefined;
                base.hemWidth = ms["裾幅"] ?? undefined;
              } else {
                // デフォルト（トップス）
                base.length = ms["着丈"] ?? undefined;
                base.bodyWidth = ms["身幅"] ?? undefined;
                base.shoulder = ms["肩幅"] ?? undefined;
                base.sleeve = ms["袖丈"] ?? undefined;
              }
            }

            return base as SizeRow;
          });

          console.log(
            "[useProductBlueprintDetail] sizeRows from measurements:",
            {
              itemType: itemTypeFromDetail,
              sizeRows,
            },
          );

          setSizes(sizeRows);

          // -------------------------------
          // modelNumbers（ModelNumber / modelNumber）
          // -------------------------------
          const modelNumberRows: ModelNumberRow[] = varsAny.map((v) => {
            const size =
              (typeof v.size === "string"
                ? v.size
                : (v.Size as string | undefined)) ?? "";

            const color =
              (typeof v.color?.name === "string"
                ? v.color.name
                : (v.Color?.Name as string | undefined)) ?? "";

            const code =
              (typeof v.modelNumber === "string"
                ? v.modelNumber
                : (v.ModelNumber as string | undefined)) ?? "";

            return { size, color, code } as ModelNumberRow;
          });
          setModelNumbers(modelNumberRows);

          // -------------------------------
          // colorRgbMap（rgb int → #rrggbb）
          // -------------------------------
          const rgbMap: Record<string, string> = {};
          varsAny.forEach((v) => {
            const name =
              (typeof v.color?.name === "string"
                ? v.color.name
                : (v.Color?.Name as string | undefined)) ?? "";

            const rgbVal =
              typeof v.color?.rgb === "number"
                ? v.color.rgb
                : typeof v.Color?.RGB === "number"
                  ? v.Color.RGB
                  : undefined;

            if (name && typeof rgbVal === "number") {
              const hex =
                "#" +
                (rgbVal >>> 0).toString(16).padStart(6, "0").toLowerCase();
              rgbMap[name] = hex;
            }
          });
          setColorRgbMap(rgbMap);
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
