import * as React from "react";
import { Tags, Trash2 } from "lucide-react";
import "./SizeVariationCard.css";

export type SizeRow = {
  id: string;
  sizeLabel: string;
  chest?: number;
  waist?: number;
  length?: number;
  shoulder?: number;
};

type SizeVariationCardProps = {
  sizes: SizeRow[];
  onRemove: (id: string) => void;
};

const SizeVariationCard: React.FC<SizeVariationCardProps> = ({ sizes, onRemove }) => {
  return (
    <section className="svc box">
      <header className="box__header">
        <Tags size={16} /> <h2 className="box__title">サイズバリエーション</h2>
      </header>

      <div className="box__body">
        <table className="svc__table">
          <thead>
            <tr>
              <th>サイズ</th>
              <th>胸囲(cm)</th>
              <th>ウエスト(cm)</th>
              <th>着丈(cm)</th>
              <th>肩幅(cm)</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {sizes.map((row) => (
              <tr key={row.id}>
                <td><input className="readonly" value={row.sizeLabel} readOnly /></td>
                <td><input className="readonly" value={row.chest ?? ""} readOnly /></td>
                <td><input className="readonly" value={row.waist ?? ""} readOnly /></td>
                <td><input className="readonly" value={row.length ?? ""} readOnly /></td>
                <td><input className="readonly" value={row.shoulder ?? ""} readOnly /></td>
                <td>
                  <button
                    className="btn btn--icon svc__remove"
                    onClick={() => onRemove(row.id)}
                    aria-label={`${row.sizeLabel} を削除`}
                  >
                    <Trash2 size={16} />
                  </button>
                </td>
              </tr>
            ))}
            {sizes.length === 0 && (
              <tr>
                <td colSpan={6} className="svc__empty">
                  登録されているサイズはありません。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
};

export default SizeVariationCard;
