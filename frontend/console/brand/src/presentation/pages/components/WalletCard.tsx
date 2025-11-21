import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardReadonly,
  CardLabel,
} from "../../../../../shell/src/shared/ui/card";

type WalletCardProps = {
  walletAddress: string;
};

export function WalletCard({ walletAddress }: WalletCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>ウォレット情報</CardTitle>
      </CardHeader>
      <CardContent>
        <CardLabel>ウォレットアドレス</CardLabel>
        <CardReadonly>
          {walletAddress?.trim() ? walletAddress : "（未設定）"}
        </CardReadonly>
      </CardContent>
    </Card>
  );
}
