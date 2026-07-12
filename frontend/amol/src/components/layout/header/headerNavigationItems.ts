// frontend/amol/src/components/layout/header/headerNavigationItems.ts
export type HeaderNavigationItem = {
  label: string;
  to: string;
};

export const publicHeaderNavigationItems: HeaderNavigationItem[] = [
  {
    label: "使い方",
    to: "/how-to-use",
  },
  {
    label: "料金プラン",
    to: "/landing#pricing",
  },
  {
    label: "会社概要",
    to: "/landing#company-overview",
  },
  {
    label: "規約・ポリシー",
    to: "/terms",
  },
  {
    label: "お問い合わせ",
    to: "/landing#contact",
  },
];