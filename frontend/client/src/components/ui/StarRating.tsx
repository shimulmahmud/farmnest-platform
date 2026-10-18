import { Star } from 'lucide-react';

interface StarRatingProps {
  rating: number;
  size?: string;
  interactive?: boolean;
  onRate?: (n: number) => void;
}

export default function StarRating({ rating, size = 'w-5 h-5', interactive = false, onRate }: StarRatingProps) {
  return (
    <div className="flex items-center gap-0.5 text-amber-400">
      {[1, 2, 3, 4, 5].map((i) => (
        <Star
          key={i}
          className={`${size} ${i <= rating ? 'fill-amber-400' : 'fill-none text-gray-300'} ${
            interactive ? 'cursor-pointer hover:text-amber-300' : ''
          }`}
          onClick={interactive && onRate ? () => onRate(i) : undefined}
          aria-label={interactive ? `Rate ${i} stars` : undefined}
        />
      ))}
    </div>
  );
}
