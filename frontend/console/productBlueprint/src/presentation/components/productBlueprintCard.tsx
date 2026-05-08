// frontend/console/productBlueprint/src/presentation/components/productBlueprintCard.tsx

import * as React from "react";
import { ShieldCheck, X, Package2 } from "lucide-react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import { Checkbox } from "../../../../shell/src/shared/ui/checkbox";

import {
  FIT_OPTIONS,
  PRODUCT_ID_TAG_OPTIONS,
  ITEM_TYPE_OPTIONS,
  type Fit,
  type ItemType,
} from "../../domain/entity/catalog";

import {
  WASH_TAG_OPTIONS,
  type WashTagOption,
} from "../../domain/entity/catalog";
import "../styles/productBlueprint.css";

type BrandOption = {
  id: string;
  name: string;
};

export type ProductBlueprintPatchInput = {
  productName?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  itemType?: string | null;
  fit?: string | null;
  material?: string | null;
  weight?: number | null;
  qualityAssurance?: string[] | null;
  productIdTag?: any;
  assigneeId?: string | null;
};

type ProductBlueprintCardProps = {
  productBlueprintPatch?: ProductBlueprintPatchInput;

  productName?: string;
  brand?: string;

  brandId?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  onChangeBrandId?: (id: string) => void;

  itemType?: ItemType;
  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;

  onChangeProductName?: (v: string) => void;
  onChangeItemType?: (v: ItemType) => void;
  onChangeFit?: (v: Fit) => void;
  onChangeMaterials?: (v: string) => void;
  onChangeWeight?: (v: number) => void;
  onChangeWashTags?: (nextTags: string[]) => void;
  onChangeProductIdTag?: (v: string) => void;

  mode?: "edit" | "view";
};

function asTrimmedString(v: unknown): string {
  return String(v ?? "").trim();
}

function normalizeProductIdTagToString(v: any): string {
  if (v === null || v === undefined) return "";
  if (typeof v === "string") return v;

  const value = (v as any)?.value;
  if (typeof value === "string") return value;

  const type = (v as any)?.type;
  if (typeof type === "string") return type;

  const Type = (v as any)?.Type;
  if (typeof Type === "string") return Type;

  const tagType = (v as any)?.tagType;
  if (typeof tagType === "string") return tagType;

  const TagType = (v as any)?.TagType;
  if (typeof TagType === "string") return TagType;

  try {
    return JSON.stringify(v);
  } catch {
    return String(v);
  }
}

function resolveOptionLabel(
  rawValue: string,
  options: Array<{ value: string; label: string }>,
): string {
  const normalized = asTrimmedString(rawValue);
  if (!normalized) return "";

  const exact = options.find((opt) => asTrimmedString(opt.value) === normalized);
  if (exact) return exact.label;

  const lower = normalized.toLowerCase();

  const lowerMatched = options.find(
    (opt) => asTrimmedString(opt.value).toLowerCase() === lower,
  );
  if (lowerMatched) return lowerMatched.label;

  if (lower === "qr" || lower === "qrcode" || lower === "qrコード") {
    const qrOpt = options.find(
      (opt) =>
        asTrimmedString(opt.value).toLowerCase() === "qrコード" ||
        asTrimmedString(opt.label).toLowerCase() === "qrコード",
    );
    if (qrOpt) return qrOpt.label;
    return "QRコード";
  }

  if (lower === "nfc") {
    const nfcOpt = options.find(
      (opt) =>
        asTrimmedString(opt.value).toLowerCase() === "nfc" ||
        asTrimmedString(opt.label).toLowerCase() === "nfc",
    );
    if (nfcOpt) return nfcOpt.label;
    return "NFC";
  }

  return normalized;
}

const ProductBlueprintCard: React.FC<ProductBlueprintCardProps> = ({
  productBlueprintPatch,

  productName,
  brand,
  brandId,
  brandOptions,
  brandLoading,
  brandError,
  onChangeBrandId,
  itemType,
  fit,
  materials,
  weight,
  washTags,
  productIdTag,
  onChangeProductName,
  onChangeItemType,
  onChangeFit,
  onChangeMaterials,
  onChangeWeight,
  onChangeWashTags,
  onChangeProductIdTag,
  mode = "edit",
}) => {
  const isEdit = mode === "edit";

  const mergedProductName = productName ?? productBlueprintPatch?.productName ?? "";
  const mergedBrandId = brandId ?? productBlueprintPatch?.brandId ?? "";
  const mergedBrandName = brand ?? productBlueprintPatch?.brandName ?? "";

  const mergedItemType =
    itemType ??
    ((typeof productBlueprintPatch?.itemType === "string"
      ? (productBlueprintPatch.itemType as ItemType)
      : undefined) as ItemType | undefined) ??
    ("" as ItemType);

  const mergedFit =
    fit ??
    ((typeof productBlueprintPatch?.fit === "string"
      ? (productBlueprintPatch.fit as Fit)
      : undefined) as Fit | undefined) ??
    ("" as Fit);

  const mergedMaterials = materials ?? productBlueprintPatch?.material ?? "";
  const mergedWeight =
    typeof weight === "number"
      ? weight
      : typeof productBlueprintPatch?.weight === "number"
        ? productBlueprintPatch.weight
        : 0;

  const mergedWashTags = Array.isArray(washTags)
    ? washTags
    : Array.isArray(productBlueprintPatch?.qualityAssurance)
      ? productBlueprintPatch.qualityAssurance.filter(
          (x): x is string => typeof x === "string" && x.trim() !== "",
        )
      : [];

  const mergedProductIdTag =
    productIdTag ?? normalizeProductIdTagToString(productBlueprintPatch?.productIdTag);

  const safeProductName = mergedProductName ?? "";
  const safeMaterials = mergedMaterials ?? "";
  const safeWeight =
    typeof mergedWeight === "number" && !Number.isNaN(mergedWeight) ? mergedWeight : 0;
  const safeWashTags = Array.isArray(mergedWashTags) ? mergedWashTags : [];
  const safeProductIdTag = mergedProductIdTag ?? "";
  const safeFit = mergedFit ?? ("" as Fit);
  const safeItemType = mergedItemType ?? ("" as ItemType);

  const displayItemType = React.useMemo(() => {
    return resolveOptionLabel(
      asTrimmedString(safeItemType),
      ITEM_TYPE_OPTIONS.map((opt) => ({
        value: asTrimmedString(opt.value),
        label: asTrimmedString(opt.label),
      })),
    );
  }, [safeItemType]);

  const displayProductIdTag = React.useMemo(() => {
    return resolveOptionLabel(
      asTrimmedString(safeProductIdTag),
      PRODUCT_ID_TAG_OPTIONS.map((opt) => ({
        value: asTrimmedString(opt.value),
        label: asTrimmedString(opt.label),
      })),
    );
  }, [safeProductIdTag]);

  const selectedBrandName =
    brandOptions?.find((b) => b.id === mergedBrandId)?.name ?? "";

  const displayBrandName =
    String(mergedBrandName ?? "").trim() ||
    String(selectedBrandName ?? "").trim() ||
    (mergedBrandId ? `(${mergedBrandId})` : "");

  const washTagGroups = React.useMemo(() => {
    const map = new Map<string, WashTagOption[]>();
    for (const opt of WASH_TAG_OPTIONS) {
      const cat = opt.category;
      const list = map.get(cat) ?? [];
      list.push(opt);
      map.set(cat, list);
    }
    return Array.from(map.entries());
  }, []);

  const handleToggleWashTag = React.useCallback(
    (value: string) => {
      if (!onChangeWashTags) return;
      if (safeWashTags.includes(value)) {
        onChangeWashTags(safeWashTags.filter((t) => t !== value));
      } else {
        onChangeWashTags([...safeWashTags, value]);
      }
    },
    [onChangeWashTags, safeWashTags],
  );

  return (
    <Card className={`pbc ${!isEdit ? "view-mode" : ""}`}>
      <CardHeader className="box__header">
        <Package2 size={16} />
        <CardTitle className="box__title">基本情報</CardTitle>
      </CardHeader>

      <CardContent className="box__body">
        <div className="label">プロダクト名</div>
        {isEdit ? (
          <Input
            value={safeProductName}
            onChange={(e) => onChangeProductName?.(e.target.value)}
            aria-label="プロダクト名"
          />
        ) : (
          <Input
            value={safeProductName}
            variant="readonly"
            readOnly
            aria-label="プロダクト名"
          />
        )}

        <div className="label">ブランド</div>
        {isEdit && brandOptions && onChangeBrandId ? (
          <div className="mb-2 space-y-1">
            <Popover>
              <PopoverTrigger>
                <Button
                  variant="outline"
                  className="w-full justify-between pbc-select-trigger"
                  aria-label="ブランドを選択"
                >
                  {selectedBrandName || "ブランドを選択してください。"}
                </Button>
              </PopoverTrigger>
              <PopoverContent align="start" className="p-1">
                {brandOptions.map((b) => (
                  <div
                    key={b.id}
                    className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                      mergedBrandId === b.id
                        ? "bg-blue-100 text-blue-700 font-medium"
                        : ""
                    }`}
                    onClick={() => onChangeBrandId(b.id)}
                  >
                    {b.name}
                  </div>
                ))}
              </PopoverContent>
            </Popover>

            {brandLoading && (
              <p className="text-xs text-slate-400">ブランドを取得中…</p>
            )}
            {brandError && (
              <p className="text-xs text-red-500">
                ブランド一覧の取得に失敗しました。
              </p>
            )}
          </div>
        ) : (
          <Input
            value={displayBrandName}
            variant="readonly"
            readOnly
            aria-label="ブランド"
          />
        )}

        <div className="pbc-fit-row">
          <div className="flex-1">
            <div className="label">アイテム種別</div>
            {isEdit ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="アイテム種別を選択"
                  >
                    {displayItemType || "アイテム種別を選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {ITEM_TYPE_OPTIONS.map(
                    (opt: { value: ItemType; label: string }) => (
                      <div
                        key={opt.value}
                        className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                          safeItemType === opt.value
                            ? "bg-blue-100 text-blue-700 font-medium"
                            : ""
                        }`}
                        onClick={() => onChangeItemType?.(opt.value)}
                      >
                        {opt.label}
                      </div>
                    ),
                  )}
                </PopoverContent>
              </Popover>
            ) : (
              <Input
                value={displayItemType}
                variant="readonly"
                readOnly
                aria-label="アイテム種別"
              />
            )}
          </div>

          <div className="flex-1">
            <div className="label">フィット</div>
            {isEdit ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="フィットを選択"
                  >
                    {safeFit || "フィットを選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {FIT_OPTIONS.map((opt) => (
                    <div
                      key={opt.value}
                      className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                        safeFit === opt.value
                          ? "bg-blue-100 text-blue-700 font-medium"
                          : ""
                      }`}
                      onClick={() => onChangeFit?.(opt.value)}
                    >
                      {opt.label}
                    </div>
                  ))}
                </PopoverContent>
              </Popover>
            ) : (
              <Input value={safeFit} variant="readonly" readOnly aria-label="フィット" />
            )}
          </div>

          <div className="flex-1">
            <div className="label">商品IDタグ</div>
            {isEdit ? (
              <Popover>
                <PopoverTrigger>
                  <Button
                    variant="outline"
                    className="w-full justify-between pbc-select-trigger"
                    aria-label="商品IDタグを選択"
                  >
                    {displayProductIdTag || "商品タグを選択してください。"}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-1">
                  {PRODUCT_ID_TAG_OPTIONS.map((opt) => (
                    <div
                      key={opt.value}
                      className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                        safeProductIdTag === opt.value
                          ? "bg-blue-100 text-blue-700 font-medium"
                          : ""
                      }`}
                      onClick={() => onChangeProductIdTag?.(opt.value)}
                    >
                      {opt.label}
                    </div>
                  ))}
                </PopoverContent>
              </Popover>
            ) : (
              <Input
                value={displayProductIdTag}
                variant="readonly"
                readOnly
                aria-label="商品IDタグ"
              />
            )}
          </div>
        </div>

        <div className="label">素材</div>
        {isEdit ? (
          <Input
            value={safeMaterials}
            onChange={(e) => onChangeMaterials?.(e.target.value)}
            aria-label="素材"
          />
        ) : (
          <Input value={safeMaterials} variant="readonly" readOnly aria-label="素材" />
        )}

        <div className="label">重さ</div>
        <div className="flex gap-8 items-center">
          {isEdit ? (
            <>
              <Input
                type="number"
                value={safeWeight}
                onChange={(e) => onChangeWeight?.(Number(e.target.value) || 0)}
                aria-label="重さ"
              />
              <span className="suffix">g</span>
            </>
          ) : (
            <>
              <Input
                value={safeWeight ? `${safeWeight}` : ""}
                variant="readonly"
                readOnly
                aria-label="重さ"
              />
              <span className="suffix">g</span>
            </>
          )}
        </div>

        <div className="label">品質保証（洗濯方法タグ）</div>
        <div className="chips flex flex-wrap gap-2">
          {safeWashTags.map((t) => (
            <Badge key={t} className="chip inline-flex items-center gap-1.5 px-2 py-1">
              <ShieldCheck size={14} />
              {t}
              {isEdit && onChangeWashTags && (
                <button
                  onClick={() => onChangeWashTags(safeWashTags.filter((x) => x !== t))}
                  className="chip-remove"
                  aria-label={`${t} を削除`}
                >
                  <X size={12} />
                </button>
              )}
            </Badge>
          ))}
        </div>

        {isEdit && onChangeWashTags && (
          <div className="mt-2 flex flex-wrap gap-2">
            {washTagGroups.map(([category, options]) => (
              <Popover key={category}>
                <PopoverTrigger>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="btn"
                    aria-label={`${category} のタグを追加`}
                  >
                    {category}
                  </Button>
                </PopoverTrigger>
                <PopoverContent align="start" className="p-2 space-y-1 w-64">
                  {options.map((opt) => {
                    const checked = safeWashTags.includes(opt.value);
                    const checkboxId = `wash-tag-${opt.value}`;
                    return (
                      <label
                        key={opt.value}
                        htmlFor={checkboxId}
                        className="flex items-center gap-2 text-sm cursor-pointer py-0.5"
                      >
                        <Checkbox
                          id={checkboxId}
                          checked={checked}
                          onCheckedChange={() => handleToggleWashTag(opt.value)}
                        />
                        <span>{opt.label}</span>
                      </label>
                    );
                  })}
                </PopoverContent>
              </Popover>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ProductBlueprintCard;