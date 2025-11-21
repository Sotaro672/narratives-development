// frontend/console/brand/src/presentation/pages/components/AssignedMemberCard.tsx
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../../shell/src/shared/ui/card";

// 共通 Table コンポーネント
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "../../../../../shell/src/shared/ui/table";

type AssignedMember = {
  id: string;
  name: string;
};

type AssignedMemberCardProps = {
  assignedMembers: AssignedMember[];
};

/**
 * ブランドに割り当てられたメンバー一覧を表示するカード
 */
export function AssignedMemberCard({ assignedMembers }: AssignedMemberCardProps) {
  const hasMembers = assignedMembers && assignedMembers.length > 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle>所属メンバー</CardTitle>
      </CardHeader>

      <CardContent>
        {!hasMembers && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            現在、所属メンバーはいません
          </div>
        )}

        {hasMembers && (
          <Table className="mt-2">
            <TableHeader>
              <TableRow>
                <TableHead>氏名</TableHead>
                <TableHead>Member ID</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {assignedMembers.map((m) => (
                <TableRow key={m.id}>
                  <TableCell>{m.name || "（未設定）"}</TableCell>
                  <TableCell>{m.id}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}
