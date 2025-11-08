// frontend/ad/mockdata.tsx

export type AdRow = {
  campaign: string;
  brand: string;
  owner: string;
  period: string; // "YYYY/M/D - YYYY/M/D"
  status: "実行中";
  spendRate: string; // "66.0%"
  spend: string; // "¥198,000"
  budget: string; // "¥300,000"
};

export const ADS: AdRow[] = [
  {
    campaign: "NEXUS Street デニムジャケット新作",
    brand: "NEXUS Street",
    owner: "渡辺 花子",
    period: "2024/3/10 - 2024/4/10",
    status: "実行中",
    spendRate: "66.0%",
    spend: "¥198,000",
    budget: "¥300,000",
  },
  {
    campaign: "LUMINA Fashion 春コレクション",
    brand: "LUMINA Fashion",
    owner: "佐藤 美咲",
    period: "2024/3/1 - 2024/3/31",
    status: "実行中",
    spendRate: "68.4%",
    spend: "¥342,000",
    budget: "¥500,000",
  },
];
