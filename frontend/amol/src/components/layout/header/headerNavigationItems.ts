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
    to: "/pricing",
  },
  {
    label: "代表者",
    to: "/faq",
  },
  {
    label: "規約・ポリシー",
    to: "/terms",
  },
  {
    label: "お問い合わせ",
    to: "/contact",
  },
];