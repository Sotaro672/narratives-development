//frontend\shell\src\layout\List\List.tsx
import "./List.css";

interface ListProps {
  /** 各ページで定義するタイトル */
  title: string;
}

export default function List({ title }: ListProps) {
  return (
    <div className="list-container">
      <h1 className="list-title">{title}</h1>
    </div>
  );
}
