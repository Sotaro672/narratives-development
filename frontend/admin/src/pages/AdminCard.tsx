// frontend/admin/src/pages/AdminCard.tsx
import { Card, CardContent, CardHeader, CardTitle } from "../../../shared/ui/card";
import { Label } from "../../../shared/ui/label";
import { Separator } from "../../../shared/ui/separator";
import { Button } from "../../../shared/ui/button";
import { Edit2 } from "lucide-react";
import "./AdminCard.css";

type Props = {
  title?: string;
  assigneeName: string;     // 例: "佐藤 美咲"
  createdByName: string;    // 例: "山田 太郎"
  createdAt: string;        // 例: "2024/1/20"
  onEditAssignee?: () => void;
  onClickAssignee?: () => void;
  onClickCreatedBy?: () => void;
};

export default function AdminCard({
  title = "管理情報",
  assigneeName,
  createdByName,
  createdAt,
  onEditAssignee,
  onClickAssignee,
  onClickCreatedBy,
}: Props) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>

      <CardContent className="space-y-4">
        {/* 担当者 */}
        <div>
          <div className="flex items-center justify-between">
            <Label className="font-semibold">担当者</Label>
            <Button
              variant="ghost"
              size="sm"
              onClick={onEditAssignee}
              aria-label="担当者を編集"
            >
              <Edit2 className="w-4 h-4" />
            </Button>
          </div>

          <p
            className={`mt-3 text-blue-600 ${
              onClickAssignee ? "cursor-pointer hover:underline" : ""
            }`}
            onClick={onClickAssignee}
          >
            {assigneeName}
          </p>
        </div>

        <Separator />

        {/* 作成者・作成日時（2カラム） */}
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            {/* v4: ユーティリティ生成していないため HSL 直指定に変更 */}
            <p className="text-sm text-[hsl(var(--muted-foreground))]">作成者</p>
            <p
              className={`text-blue-600 ${
                onClickCreatedBy ? "cursor-pointer hover:underline" : ""
              }`}
              onClick={onClickCreatedBy}
            >
              {createdByName}
            </p>
          </div>

          <div className="space-y-2">
            <p className="text-sm text-[hsl(var(--muted-foreground))]">作成日時</p>
            <p className="text-base">{createdAt}</p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

