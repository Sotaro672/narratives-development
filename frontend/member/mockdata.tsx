// frontend/member/mockdata.tsx

export type MemberRow = {
  name: string;
  email: string;
  role: string;
  brand: string[];
  taskCount: number;
  permissionCount: number;
  registeredAt: string;
};

export const MEMBERS: MemberRow[] = [
  {
    name: "小林 静香",
    email: "designer.lumina@narratives.com",
    role: "生産設計責任者",
    brand: ["LUMINA Fashion"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/6/25",
  },
  {
    name: "渡辺 花子",
    email: "support.lumina@narratives.com",
    role: "問い合わせ担当者",
    brand: ["LUMINA Fashion"],
    taskCount: 1,
    permissionCount: 2,
    registeredAt: "2024/6/15",
  },
  {
    name: "中村 拓也",
    email: "token.lumina@narratives.com",
    role: "トークン管理者",
    brand: ["LUMINA Fashion"],
    taskCount: 0,
    permissionCount: 3,
    registeredAt: "2024/4/18",
  },
  {
    name: "伊藤 愛子",
    email: "support.nexus@narratives.com",
    role: "問い合わせ担当者",
    brand: ["NEXUS Street"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/4/5",
  },
  {
    name: "田中 雄太",
    email: "marketing.nexus@narratives.com",
    role: "ブランド管理者",
    brand: ["NEXUS Street"],
    taskCount: 1,
    permissionCount: 4,
    registeredAt: "2024/3/22",
  },
  {
    name: "高橋 健太",
    email: "token.nexus@narratives.com",
    role: "トークン管理者",
    brand: ["NEXUS Street"],
    taskCount: 7,
    permissionCount: 3,
    registeredAt: "2024/3/10",
  },
  {
    name: "松本 葵",
    email: "designer.nexus@narratives.com",
    role: "生産設計責任者",
    brand: ["NEXUS Street"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/3/5",
  },
  {
    name: "佐藤 美咲",
    email: "manager.lumina@narratives.com",
    role: "ブランド管理者",
    brand: ["LUMINA Fashion"],
    taskCount: 10,
    permissionCount: 4,
    registeredAt: "2024/2/20",
  },
  {
    name: "山田 太郎",
    email: "admin@narratives.com",
    role: "管理者",
    brand: ["LUMINA Fashion", "NEXUS Street"],
    taskCount: 5,
    permissionCount: 12,
    registeredAt: "2024/1/15",
  },
];
