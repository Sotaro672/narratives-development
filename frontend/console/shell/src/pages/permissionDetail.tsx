import PageStyle from "../layout/PageStyle/PageStyle";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardLabel,
  CardReadonly,
} from "../shared/ui/card";

import { usePermissionDetail } from "../features/permission/presentation/hook/usePermissionDetail";

export default function PermissionDetail() {
  const {
    permission,
    loading,
    error,
    title,
    handleBack,
  } = usePermissionDetail();

  if (loading) {
    return (
      <PageStyle
        layout="single"
        title={title}
        onBack={handleBack}
      >
        <div className="text-sm text-[hsl(var(--muted-foreground))]">
          読み込み中...
        </div>
      </PageStyle>
    );
  }

  if (error) {
    return (
      <PageStyle
        layout="single"
        title={title}
        onBack={handleBack}
      >
        <div className="text-sm text-red-600">
          {error}
        </div>
      </PageStyle>
    );
  }

  if (!permission) {
    return (
      <PageStyle
        layout="single"
        title={title}
        onBack={handleBack}
      >
        <div className="text-sm text-[hsl(var(--muted-foreground))]">
          権限情報が見つかりません。
        </div>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="single"
      title={title}
      onBack={handleBack}
    >
      <div className="space-y-4 max-w-3xl">
        <Card>
          <CardHeader>
            <CardTitle>基本情報</CardTitle>
          </CardHeader>

          <CardContent>
            <CardLabel>権限ID</CardLabel>
            <CardReadonly>{permission.id}</CardReadonly>

            <CardLabel>権限名</CardLabel>
            <CardReadonly>{permission.name}</CardReadonly>

            <CardLabel>カテゴリ</CardLabel>
            <CardReadonly>{permission.category}</CardReadonly>

            <CardLabel>説明</CardLabel>
            <div className="mt-1 rounded-lg border px-3 py-2 text-sm bg-[hsl(var(--muted-bg))] text-[hsl(var(--muted-foreground))]">
              {permission.description}
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}