// frontend/console/list/src/presentation/pages/listCreate.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { useListCreate } from "../hook/useListCreate";

export default function ListCreate() {
  const { onBack, inventoryId } = useListCreate();

  return (
    <PageStyle
      layout="grid-2"
      title={inventoryId ? `出品の作成（inventoryId: ${inventoryId}）` : "出品の作成"}
      onBack={onBack}
    >
      {/* 左カラム：空（grid-2 のレイアウト維持） */}
      <div />

      {/* 右カラム：空（grid-2 のレイアウト維持） */}
      <div />
    </PageStyle>
  );
}
