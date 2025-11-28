// frontend/console/log/presentation/LogCard.tsx
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shell/src/shared/ui";
import "../style/logStyle.css";

export type LogCardEntry = {
  /** 一意な ID（履歴 ID / ログ ID など） */
  id: string;

  /** 自由形式のメッセージ（従来の LogCard 用） */
  message?: string;

  /** 従来の createdAt（文字列フォーマット済み） */
  createdAt?: string;

  /** 履歴バージョン番号（商品設計履歴などで使用） */
  version?: number;

  /** 更新日時（履歴用）。あればこちらを優先して表示 */
  updatedAt?: string;

  /** 更新者の表示名（フロント側で Member 名を解決したもの） */
  updatedByName?: string;
};

export type LogCardProps = {
  /** カードタイトル（デフォルト: "ログ"） */
  title?: string;

  /** 表示するログ / 履歴の配列 */
  logs?: LogCardEntry[];

  /** ログが空の場合のメッセージ */
  emptyText?: string;
};

export default function LogCard({
  title = "ログ",
  logs = [],
  emptyText = "ログはまだありません。",
}: LogCardProps) {
  return (
    <Card className="log-card">
      <CardHeader className="log-card__header">
        <CardTitle className="log-card__title">{title}</CardTitle>
      </CardHeader>

      <CardContent>
        {logs.length === 0 ? (
          <p className="text-sm text-muted-foreground">{emptyText}</p>
        ) : (
          logs.map((log) => {
            // 1行目: version があれば "v3 〜" のように表示し、それ以外は message をそのまま
            const hasVersion = typeof log.version === "number";
            const primaryText = hasVersion
              ? `v${log.version}${
                  log.message ? ` - ${log.message}` : ""
                }`
              : log.message ?? "";

            // 2行目: updatedAt / createdAt / updatedByName を組み合わせて表示
            const timestamp = log.updatedAt ?? log.createdAt ?? "";
            const hasActor = !!log.updatedByName;

            return (
              <div key={log.id} className="log-card__item">
                <div className="log-card__item-main text-sm">{primaryText}</div>

                {(timestamp || hasActor) && (
                  <div className="text-xs text-muted-foreground mt-1 log-card__item-meta">
                    {timestamp && <span>{timestamp}</span>}
                    {timestamp && hasActor && <span> ・ </span>}
                    {hasActor && <span>{log.updatedByName}</span>}
                  </div>
                )}
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}
