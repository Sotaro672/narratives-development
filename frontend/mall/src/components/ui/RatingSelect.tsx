// frontend/mall/src/components/ui/RatingSelect.tsx

import { useState } from "react";
import { Star } from "lucide-react";

import "./ratingSelect.css";

type RatingSelectProps = {
  value: number;
  onChange: (rating: number) => void;
  min?: number;
  max?: number;
  descending?: boolean;
  className?: string;
  disabled?: boolean;
  ariaLabel?: string;
};

function createRatings(min: number, max: number): number[] {
  const ratings: number[] = [];

  for (let value = min; value <= max; value += 1) {
    ratings.push(value);
  }

  return ratings;
}

export default function RatingSelect(props: RatingSelectProps) {
  const {
    value,
    onChange,
    min = 1,
    max = 5,
    className,
    disabled = false,
    ariaLabel = "評価",
  } = props;

  const [hoveredRating, setHoveredRating] = useState<number | null>(null);
  const ratings = createRatings(min, max);
  const displayRating = hoveredRating ?? value;

  const handleKeyDown = (
    event: React.KeyboardEvent<HTMLButtonElement>,
    rating: number,
  ) => {
    if (disabled) return;

    if (event.key === "ArrowRight" || event.key === "ArrowUp") {
      event.preventDefault();
      onChange(Math.min(max, rating + 1));
      return;
    }

    if (event.key === "ArrowLeft" || event.key === "ArrowDown") {
      event.preventDefault();
      onChange(Math.max(min, rating - 1));
    }
  };

  return (
    <div
      className={["ui-rating-select", className].filter(Boolean).join(" ")}
      role="radiogroup"
      aria-label={ariaLabel}
      onMouseLeave={() => setHoveredRating(null)}
    >
      {ratings.map((rating) => {
        const active = rating <= displayRating;

        return (
          <button
            key={rating}
            type="button"
            className={[
              "ui-rating-select__button",
              active ? "ui-rating-select__button--active" : "",
            ]
              .filter(Boolean)
              .join(" ")}
            role="radio"
            aria-checked={value === rating}
            aria-label={`${rating}つ星`}
            title={`${rating}つ星`}
            disabled={disabled}
            onMouseEnter={() => setHoveredRating(rating)}
            onFocus={() => setHoveredRating(rating)}
            onBlur={() => setHoveredRating(null)}
            onClick={() => onChange(rating)}
            onKeyDown={(event) => handleKeyDown(event, rating)}
          >
            <Star
              className="ui-rating-select__star"
              fill={active ? "currentColor" : "none"}
              strokeWidth={1.8}
              aria-hidden="true"
            />
          </button>
        );
      })}
    </div>
  );
}