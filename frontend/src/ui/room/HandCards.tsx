import { CardTile } from "./CardTile";

interface HandCardsProps {
  cards: string[];
  selectedCards: string[];
  disabled?: boolean;
  onToggle: (card: string) => void;
}

export function HandCards({ cards, selectedCards, disabled = false, onToggle }: HandCardsProps) {
  return (
    <div className="hand-cards" aria-label="我的手牌">
      {cards.map((card, index) => (
        <CardTile
          key={`${card}-${index}`}
          code={card}
          selected={selectedCards.includes(card)}
          disabled={disabled}
          onClick={() => onToggle(card)}
        />
      ))}
    </div>
  );
}
