interface CardTileProps {
  code: string;
  selected?: boolean;
  disabled?: boolean;
  faceDown?: boolean;
  compact?: boolean;
  onClick?: () => void;
}

const redSuits = new Set(["H", "D"]);

export function CardTile({
  code,
  selected = false,
  disabled = false,
  faceDown = false,
  compact = false,
  onClick,
}: CardTileProps) {
  const content = parseCardCode(code);
  const isRed = content ? redSuits.has(content.suit) || content.rank === "BJ" || content.rank === "RJ" : false;

  return (
    <button
      className={`card-tile${selected ? " is-selected" : ""}${compact ? " is-compact" : ""}${
        faceDown ? " is-face-down" : ""
      }`}
      type="button"
      disabled={disabled || faceDown}
      onClick={onClick}
      aria-label={faceDown ? "背面牌" : `牌 ${code}`}
    >
      {faceDown ? (
        <span className="card-tile__back" />
      ) : (
        <>
          <span className={`card-tile__rank${isRed ? " is-red" : ""}`}>{content?.rank ?? code}</span>
          <span className={`card-tile__suit${isRed ? " is-red" : ""}`}>{renderSuit(content?.suit)}</span>
        </>
      )}
    </button>
  );
}

function parseCardCode(code: string) {
  if (code === "BJ" || code === "RJ") {
    return { suit: "", rank: code };
  }
  if (code.length < 2) {
    return null;
  }
  return {
    suit: code.slice(0, 1),
    rank: code.slice(1),
  };
}

function renderSuit(suit?: string) {
  switch (suit) {
    case "S":
      return "♠";
    case "H":
      return "♥";
    case "C":
      return "♣";
    case "D":
      return "♦";
    default:
      return "J";
  }
}
