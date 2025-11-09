// frontend/permission/src/pages/permissionDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardReadonly,
} from "../../../shared/ui/card";

/**
 * 権限詳細ページ
 * layout="single" + PageStyle ヘッダー一体型
 */
export default function PermissionDetail() {
  const navigate = useNavigate();
  const { permissionId } = useParams<{ permissionId: string }>();

  // モックデータ（API接続前）
  const [permission] = React.useState({
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
  });

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="single"
      title={`権限詳細：${permission.name}`}
      onBack={handleBack}
    >
      <div className="space-y-4 max-w-3xl">
        {/* 基本情報 */}
        <Card>
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>
          <CardContent>
            <CardLabel>権限名</CardLabel>
            <CardReadonly>{permission.name}</CardReadonly>

            <CardLabel>権限コード</CardLabel>
            <CardReadonly>{permission.code}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="mt-1 rounded-lg border px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {permission.description}
            </div>
          </CardContent>
        </Card>

        {/* スコープ一覧 */}
        <Card>
          <CardHeader>
            <CardTitle>付与される操作範囲（スコープ）</CardTitle>
          </CardHeader>
          <CardContent>
            {permission.scopes.length === 0 ? (
              <div className="text-xs text-[hsl(var(--muted-foreground))]">
                スコープは設定されていません。
              </div>
            ) : (
              <ul className="list-disc pl-5 space-y-1 text-sm">
                {permission.scopes.map((s) => (
                  <li key={s}>{s}</li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* 紐づくメンバー */}
        <Card>
          <CardHeader>
            <CardTitle>この権限を持つメンバー</CardTitle>
          </CardHeader>
          <CardContent>
            {permission.assignedMembers.length === 0 ? (
              <div className="text-xs text-[hsl(var(--muted-foreground))]">
                現在この権限を持つメンバーはいません。
              </div>
            ) : (
              <ul className="list-none space-y-1 text-sm">
                {permission.assignedMembers.map((m) => (
                  <li
                    key={m}
                    className="inline-flex items-center px-2 py-1 mr-2 mb-2 rounded-full bg-slate-50 text-slate-700 text-xs"
                  >
                    {m}
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        {/* メタ情報 */}
        <Card>
          <CardHeader>
            <CardTitle>メタ情報</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
            <div>
              <CardLabel>作成日</CardLabel>
              <CardReadonly>{permission.createdAt}</CardReadonly>
            </div>
            <div>
              <CardLabel>最終更新日</CardLabel>
              <CardReadonly>{permission.updatedAt}</CardReadonly>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
