// frontend/amol/src/components/ui/RatingSelect.tsx
import "./rating-select.css";

type RatingSelectProps = {
  value: number;
  onChange: (rating: number) => void;
  min?: number;
  max?: number;
  descending?: boolean;
  className?: string;
};

function createRatings(min: number, max: number, descending: boolean): number[] {
  const out: number[] = [];

  for (let value = min; value <= max; value += 1) {
    out.push(value);
  }

  return descending ? out.reverse() : out;
}

export default function RatingSelect(props: RatingSelectProps) {
  const {
    value,
    onChange,
    min = 1,
    max = 5,
    descending = true,
    className,
  } = props;

  const ratings = createRatings(min, max, descending);

  return (
    <select
      className={["ui-rating-select", className].filter(Boolean).join(" ")}
      value={value}
      onChange={(event) => onChange(Number(event.target.value))}
    >
      {ratings.map((rating) => (
        <option value={rating} key={rating}>
          {rating}
        </option>
      ))}
    </select>
  );
}