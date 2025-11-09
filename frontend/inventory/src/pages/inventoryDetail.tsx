import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import ProductBlueprintCard from "../../../productBlueprint/src/pages/productBlueprintCard";
import TokenBlueprintCard from "../../../tokenBlueprint/src/pages/tokenBlueprintCard";
import InventoryCard, {
  type InventoryRow,
} from "./inventoryCard";

type Fit =
  | "レギュラーフィット"
  | "スリムフィット"
  | "リラックスフィット"
  | "オーバーサイズ";

export default function InventoryDetail() {
  const navigate = useNavigate();
  const { inventoryId } = useParams<{ inventoryId: string }>();

  // ─────────────────────────────────────────
  // モックデータ
  // ─────────────────────────────────────────
  const [productName] = React.useState("シルクブラウス プレミアムライン");
  const [brand] = React.useState("LUMINA Fashion");
  const [fit] = React.useState<Fit>("レギュラーフィット");
  const [materials] = React.useState("シルク100%、裏地:ポリエステル100%");
  const [weight] = React.useState<number>(180);
  const [washTags] = React.useState<string[]>([
    "手洗い",
    "ドライクリーニング",
    "陰干し",
  ]);
  const [productIdTag] = React.useState("QRコード");

  // 在庫情報
  const [token] = React.useState("LUMINA VIP Token");
  const [total] = React.useState(221);

  // モデル別在庫データ
  const [inventoryRows] = React.useState<InventoryRow[]>([
    { modelCode: "LM-SB-S-WHT", size: "S", colorName: "ホワイト", colorCode: "#ffffff", stock: 25 },
    { modelCode: "LM-SB-M-WHT", size: "M", colorName: "ホワイト", colorCode: "#ffffff", stock: 42 },
    { modelCode: "LM-SB-L-WHT", size: "L", colorName: "ホワイト", colorCode: "#ffffff", stock: 38 },
    { modelCode: "LM-SB-M-BLK", size: "M", colorName: "ブラック", colorCode: "#000000", stock: 35 },
    { modelCode: "LM-SB-L-BLK", size: "L", colorName: "ブラック", colorCode: "#000000", stock: 28 },
    { modelCode: "LM-SB-M-NVY", size: "M", colorName: "ネイビー", colorCode: "#1e3a8a", stock: 31 },
    { modelCode: "LM-SB-L-NVY", size: "L", colorName: "ネイビー", colorCode: "#1e3a8a", stock: 22 },
  ]);

  // 管理情報
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("佐藤 美咲");
  const [createdAt] = React.useState("2024/3/15");

  // 戻るボタン
  const onBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 閲覧専用ハンドラ
  const noop = () => {};
  const noopStr = (_v: string) => {};

  return (
    <PageStyle
      layout="grid-2"
      title={`在庫詳細：${inventoryId ?? "不明ID"}`}
      onBack={onBack}
      onSave={undefined}
    >
      {/* 左カラム */}
      <div>
        {/* 商品基本情報カード（閲覧モード） */}
        <ProductBlueprintCard
          mode="view"
          productName={productName}
          brand={brand}
          fit={fit}
          materials={materials}
          weight={weight}
          washTags={washTags}
          productIdTag={productIdTag}
          onChangeProductName={noopStr}
          onChangeFit={noop as (v: Fit) => void}
          onChangeMaterials={noopStr}
          onChangeWeight={noop as (v: number) => void}
          onChangeWashTags={noop as (next: string[]) => void}
          onChangeProductIdTag={noopStr}
        />

        {/* トークン設計カード（閲覧モード） */}
        <div style={{ marginTop: 16 }}>
          <TokenBlueprintCard />
        </div>

        {/* モデル別在庫カード */}
        <div style={{ marginTop: 16 }}>
          <InventoryCard rows={inventoryRows} />
        </div>
      </div>

      {/* 右カラム（管理情報） */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
