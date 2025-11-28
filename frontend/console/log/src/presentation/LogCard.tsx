// frontend/console/log/presentation/LogCard.tsx

import * as React from "react";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../shell/src/shared/ui";
import "../style/logStyle.css";

export type LogCardProps = {
  logs?: Array<{
    id: string;
    message: string;
    createdAt: string;
  }>;
};

export default function LogCard({ logs = [] }: LogCardProps) {
  return (
    <Card className="log-card">
      <CardHeader>
        <CardTitle className="log-card__title">ログ</CardTitle>
      </CardHeader>

      <CardContent>
        {logs.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            ログはまだありません。
          </p>
        ) : (
          logs.map((log) => (
            <div key={log.id} className="log-card__item">
              <div>{log.message}</div>
              <div className="text-xs text-muted-foreground mt-1">
                {log.createdAt}
              </div>
            </div>
          ))
        )}
      </CardContent>
    </Card>
  );
}
