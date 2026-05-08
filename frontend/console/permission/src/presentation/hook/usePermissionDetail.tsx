// frontend/console/permission/src/presentation/hook/usePermissionDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";

/** 表示用の権限詳細モデル（ページで使用している項目のみ） */
export type PermissionDetailModel = {
  id: string;
  name: string;
  code: string;
  description: string;
  scopes: string[];
  assignedMembers: string[];
  createdAt: string;
  updatedAt: string;
};

function buildMock(permissionId?: string | null): PermissionDetailModel {
  return {
    id: permissionId ?? "perm_brand_admin",
    name: "ブランド管理者権限",
    code: "BRAND_ADMIN",
    description:
      "ブランド情報の閲覧・編集、ブランドに紐づくメンバーやキャンペーン設定を行うことができます。",
    scopes: [
      "ブランド情報の閲覧/編集",
      "メンバーのロール割り当て",
      "キャンペーン設定の閲覧",
      "通知設定の更新",
    ],
    assignedMembers: ["山田 太郎", "佐藤 美咲"],
    createdAt: "2024/01/15",
    updatedAt: "2025/10/30",
  };
}

/**
 * 権限詳細ページ用のロジック（ルーティング/戻る操作/データ保持）
 * - いまはモックデータ。API 接続時はここで fetch → setPermission に置き換えます。
 */
export function usePermissionDetail() {
  const navigate = useNavigate();
  const { permissionId } = useParams<{ permissionId: string }>();

  // 取得データ（モック）
  const [permission, setPermission] = React.useState<PermissionDetailModel>(
    () => buildMock(permissionId)
  );

  // API接続時の例：
  // React.useEffect(() => {
  //   let aborted = false;
  //   (async () => {
  //     const res = await fetch(`/api/permissions/${encodeURIComponent(permissionId ?? "")}`);
  //     if (!aborted && res.ok) {
  //       const data = (await res.json()) as PermissionDetailModel;
  //       setPermission(data);
  //     }
  //   })();
  //   return () => {
  //     aborted = true;
  //   };
  // }, [permissionId]);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const title = `権限詳細：${permission.name}`;

  return {
    permission,
    setPermission, // 必要なら編集機能で利用
    handleBack,
    title,
  };
}

export default usePermissionDetail;
