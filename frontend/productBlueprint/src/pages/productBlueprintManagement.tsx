//frontend\productBlueprint\src\pages\productBlueprintManagement.tsx
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

export default function ProductBlueprintManagement() {
  const headers = [
    "プロダクト",
    <>
      <span>ブランド</span>
      <button className="lp-th-filter" aria-label="ブランドで絞り込む">
        <Filter size={16} />
      </button>
    </>,
    <>
      <span>担当者</span>
      <button className="lp-th-filter" aria-label="担当者で絞り込む">
        <Filter size={16} />
      </button>
    </>,
    <>
      <span>商品ID</span>
      <button className="lp-th-filter" aria-label="商品IDで絞り込む">
        <Filter size={16} />
      </button>
    </>,
    "作成日",
    "作成日",
  ];

  return (
    <List
      title="商品設計"
      headerCells={headers}
      showCreateButton
      createLabel="商品設計を作成"
      onCreate={() => console.log("create")}
      showResetButton
      onReset={() => console.log("reset")}
    >
      <tr>
        <td>シルクブラウス プレミアムライン</td>
        <td><span className="lp-brand-pill">LUMINA Fashion</span></td>
        <td>佐藤 美咲</td>
        <td>QR</td>
        <td>2024/1/15</td>
        <td>2024/1/15</td>
      </tr>
      <tr>
        <td>デニムジャケット ヴィンテージ加工</td>
        <td><span className="lp-brand-pill">NEXUS Street</span></td>
        <td>高橋 健太</td>
        <td>QR</td>
        <td>2024/1/10</td>
        <td>2024/1/10</td>
      </tr>
    </List>
  );
}
