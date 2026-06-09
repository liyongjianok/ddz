import type { RoomSnapshotPlay } from "../types/api";

export type SortMode = "rank" | "suit";

export interface ParsedCard {
  code: string;
  suit: string;
  rank: string;
  rankValue: number;
  suitValue: number;
}

export interface ClientCardGroup {
  type: string;
  rank: string;
  rankValue: number;
  length: number;
  cards: string[];
}

const rankValues: Record<string, number> = {
  "3": 1,
  "4": 2,
  "5": 3,
  "6": 4,
  "7": 5,
  "8": 6,
  "9": 7,
  T: 8,
  J: 9,
  Q: 10,
  K: 11,
  A: 12,
  "2": 13,
  BJ: 14,
  RJ: 15,
};

const suitValues: Record<string, number> = {
  S: 4,
  H: 3,
  C: 2,
  D: 1,
  "": 0,
};

const straightMaxRank = rankValues.A;

export function parseCard(code: string): ParsedCard | null {
  if (code === "BJ" || code === "RJ") {
    return {
      code,
      suit: "",
      rank: code,
      rankValue: rankValues[code],
      suitValue: suitValues[""],
    };
  }

  if (code.length !== 2) {
    return null;
  }

  const suit = code.slice(0, 1);
  const rank = code.slice(1);
  if (!(rank in rankValues) || !(suit in suitValues)) {
    return null;
  }

  return {
    code,
    suit,
    rank,
    rankValue: rankValues[rank],
    suitValue: suitValues[suit],
  };
}

export function sortCards(cards: string[], mode: SortMode) {
  return [...cards].sort((left, right) => {
    const a = parseCard(left);
    const b = parseCard(right);
    if (!a || !b) {
      return left.localeCompare(right);
    }

    if (mode === "suit" && a.suitValue !== b.suitValue) {
      return b.suitValue - a.suitValue;
    }

    if (a.rankValue !== b.rankValue) {
      return b.rankValue - a.rankValue;
    }
    return b.suitValue - a.suitValue;
  });
}

export function recognizeCards(cards: string[]): ClientCardGroup | null {
  const parsed = cards.map(parseCard);
  if (cards.length === 0 || parsed.some((card) => !card)) {
    return null;
  }

  const validCards = parsed as ParsedCard[];
  const ranks = countByRank(validCards);
  const rankItems = [...ranks.entries()].sort((a, b) => a[0] - b[0]);

  if (cards.length === 2 && hasRank(validCards, "BJ") && hasRank(validCards, "RJ")) {
    return buildGroup("rocket", "RJ", 2, cards);
  }

  switch (cards.length) {
    case 1:
      return buildGroup("single", validCards[0].rank, 1, cards);
    case 2:
      if (rankItems.length === 1) {
        return buildGroup("pair", validCards[0].rank, 2, cards);
      }
      break;
    case 3:
      if (rankItems.length === 1) {
        return buildGroup("triple", validCards[0].rank, 3, cards);
      }
      break;
    case 4: {
      const triple = rankItems.find(([, count]) => count === 3);
      const bomb = rankItems.find(([, count]) => count === 4);
      if (bomb) {
        return buildGroup("bomb", valueToRank(bomb[0]), 4, cards);
      }
      if (triple) {
        return buildGroup("triple_with_single", valueToRank(triple[0]), 4, cards);
      }
      break;
    }
    case 5: {
      const triple = rankItems.find(([, count]) => count === 3);
      const pair = rankItems.find(([, count]) => count === 2);
      if (triple && pair) {
        return buildGroup("triple_with_pair", valueToRank(triple[0]), 5, cards);
      }
      break;
    }
  }

  const straight = recognizeStraight(rankItems, cards);
  if (straight) {
    return straight;
  }

  const pairStraight = recognizeRepeatedStraight(rankItems, cards, 2, "pair_straight");
  if (pairStraight) {
    return pairStraight;
  }

  const airplane = recognizeRepeatedStraight(rankItems, cards, 3, "airplane");
  if (airplane) {
    return airplane;
  }

  const airplaneWithSingles = recognizeAirplaneWithAttachments(rankItems, cards, false);
  if (airplaneWithSingles) {
    return airplaneWithSingles;
  }

  const airplaneWithPairs = recognizeAirplaneWithAttachments(rankItems, cards, true);
  if (airplaneWithPairs) {
    return airplaneWithPairs;
  }

  const fourWithSingles = recognizeFourWithAttachments(rankItems, cards, false);
  if (fourWithSingles) {
    return fourWithSingles;
  }

  const fourWithPairs = recognizeFourWithAttachments(rankItems, cards, true);
  if (fourWithPairs) {
    return fourWithPairs;
  }

  return null;
}

export function canPlaySelection(cards: string[], previous: RoomSnapshotPlay | undefined, mySeat: number) {
  const group = recognizeCards(cards);
  if (!group) {
    return false;
  }

  if (!previous || previous.seat_index === mySeat) {
    return true;
  }

  return canBeat(group, previousToGroup(previous));
}

export function describeSelection(cards: string[], previous: RoomSnapshotPlay | undefined, mySeat: number) {
  if (cards.length === 0) {
    return "请选择要出的牌";
  }

  const group = recognizeCards(cards);
  if (!group) {
    return "当前选牌不是合法牌型";
  }

  if (previous && previous.seat_index !== mySeat && !canBeat(group, previousToGroup(previous))) {
    return `${renderGroupType(group.type)} 压不过上一手`;
  }

  return `${renderGroupType(group.type)}，可以出`;
}

export function findHintCards(hand: string[], previous: RoomSnapshotPlay | undefined, mySeat: number) {
  const candidates = generateCandidateGroups(hand);
  const playable = candidates
    .filter((group) => {
      if (!previous || previous.seat_index === mySeat) {
        return true;
      }
      return canBeat(group, previousToGroup(previous));
    })
    .sort((a, b) => {
      if (a.type === "rocket" && b.type !== "rocket") {
        return 1;
      }
      if (a.type !== "rocket" && b.type === "rocket") {
        return -1;
      }
      if (a.type === "bomb" && b.type !== "bomb") {
        return 1;
      }
      if (a.type !== "bomb" && b.type === "bomb") {
        return -1;
      }
      return a.rankValue - b.rankValue || a.cards.length - b.cards.length;
    });

  return playable[0]?.cards ?? [];
}

export function renderGroupType(type: string) {
  switch (type) {
    case "single":
      return "单牌";
    case "pair":
      return "对子";
    case "triple":
      return "三张";
    case "triple_with_single":
      return "三带一";
    case "triple_with_pair":
      return "三带一对";
    case "straight":
      return "顺子";
    case "pair_straight":
      return "连对";
    case "airplane":
      return "飞机";
    case "airplane_with_singles":
      return "飞机带单";
    case "airplane_with_pairs":
      return "飞机带对";
    case "four_with_two_singles":
      return "四带二";
    case "four_with_two_pairs":
      return "四带两对";
    case "bomb":
      return "炸弹";
    case "rocket":
      return "王炸";
    default:
      return "牌型";
  }
}

function generateCandidateGroups(hand: string[]) {
  const sorted = sortCards(hand, "rank");
  const parsed = sorted.map(parseCard).filter(Boolean) as ParsedCard[];
  const cardsByRank = groupCardsByRank(parsed);
  const candidates: ClientCardGroup[] = [];

  for (const card of parsed) {
    candidates.push(buildGroup("single", card.rank, 1, [card.code]));
  }

  for (const [rankValue, cards] of cardsByRank) {
    const rank = valueToRank(rankValue);
    if (cards.length >= 2) {
      candidates.push(buildGroup("pair", rank, 2, cards.slice(0, 2).map((card) => card.code)));
    }
    if (cards.length >= 3) {
      candidates.push(buildGroup("triple", rank, 3, cards.slice(0, 3).map((card) => card.code)));
    }
    if (cards.length >= 4) {
      candidates.push(buildGroup("bomb", rank, 4, cards.slice(0, 4).map((card) => card.code)));
    }
  }

  const jokerCards = parsed.filter((card) => card.rank === "BJ" || card.rank === "RJ");
  if (jokerCards.length === 2) {
    candidates.push(buildGroup("rocket", "RJ", 2, jokerCards.map((card) => card.code)));
  }

  candidates.push(...generateStraights(cardsByRank, 1, 5, "straight"));
  candidates.push(...generateStraights(cardsByRank, 2, 3, "pair_straight"));
  candidates.push(...generateStraights(cardsByRank, 3, 2, "airplane"));

  return dedupeGroups(candidates);
}

function generateStraights(
  cardsByRank: Map<number, ParsedCard[]>,
  repeatCount: number,
  minRanks: number,
  type: string,
) {
  const ranks = [...cardsByRank.keys()]
    .filter((rank) => rank <= straightMaxRank && (cardsByRank.get(rank)?.length ?? 0) >= repeatCount)
    .sort((a, b) => a - b);
  const groups: ClientCardGroup[] = [];

  for (let start = 0; start < ranks.length; start++) {
    const sequence = [ranks[start]];
    for (let next = start + 1; next < ranks.length; next++) {
      if (ranks[next] !== sequence[sequence.length - 1] + 1) {
        break;
      }
      sequence.push(ranks[next]);
      if (sequence.length >= minRanks) {
        const cards = sequence.flatMap((rank) => (cardsByRank.get(rank) ?? []).slice(0, repeatCount).map((card) => card.code));
        groups.push(buildGroup(type, valueToRank(sequence[sequence.length - 1]), sequence.length, cards));
      }
    }
  }

  return groups;
}

function canBeat(candidate: ClientCardGroup, previous: ClientCardGroup) {
  if (previous.type === "rocket") {
    return false;
  }
  if (candidate.type === "rocket") {
    return true;
  }
  if (candidate.type === "bomb") {
    return previous.type !== "bomb" || candidate.rankValue > previous.rankValue;
  }
  if (previous.type === "bomb") {
    return false;
  }
  return candidate.type === previous.type && candidate.length === previous.length && candidate.rankValue > previous.rankValue;
}

function previousToGroup(previous: RoomSnapshotPlay): ClientCardGroup {
  return {
    type: previous.card_group.type,
    rank: previous.card_group.rank,
    rankValue: rankValues[previous.card_group.rank] ?? 0,
    length: previous.card_group.length,
    cards: previous.cards,
  };
}

function recognizeStraight(rankItems: Array<[number, number]>, cards: string[]) {
  if (cards.length < 5 || rankItems.length !== cards.length || rankItems.some(([, count]) => count !== 1)) {
    return null;
  }
  if (!isConsecutive(rankItems.map(([rank]) => rank)) || rankItems[rankItems.length - 1][0] > straightMaxRank) {
    return null;
  }
  const primary = rankItems[rankItems.length - 1][0];
  return buildGroup("straight", valueToRank(primary), cards.length, cards);
}

function recognizeRepeatedStraight(rankItems: Array<[number, number]>, cards: string[], count: number, type: string) {
  if (rankItems.length < 2 || cards.length !== rankItems.length * count || rankItems.some(([, itemCount]) => itemCount !== count)) {
    return null;
  }
  if (!isConsecutive(rankItems.map(([rank]) => rank)) || rankItems[rankItems.length - 1][0] > straightMaxRank) {
    return null;
  }
  const primary = rankItems[rankItems.length - 1][0];
  return buildGroup(type, valueToRank(primary), rankItems.length, cards);
}

function recognizeAirplaneWithAttachments(rankItems: Array<[number, number]>, cards: string[], withPairs: boolean) {
  const tripleRanks = rankItems.filter(([, count]) => count === 3).map(([rank]) => rank);
  if (tripleRanks.length < 2 || !isConsecutive(tripleRanks) || tripleRanks[tripleRanks.length - 1] > straightMaxRank) {
    return null;
  }

  const expectedLength = tripleRanks.length * (withPairs ? 5 : 4);
  if (cards.length !== expectedLength) {
    return null;
  }

  const attachmentItems = rankItems.filter(([rank]) => !tripleRanks.includes(rank));
  if (attachmentItems.length !== tripleRanks.length) {
    return null;
  }

  const attachmentsValid = attachmentItems.every(([, count]) => count === (withPairs ? 2 : 1));
  if (!attachmentsValid) {
    return null;
  }

  const primary = tripleRanks[tripleRanks.length - 1];
  return buildGroup(withPairs ? "airplane_with_pairs" : "airplane_with_singles", valueToRank(primary), tripleRanks.length, cards);
}

function recognizeFourWithAttachments(rankItems: Array<[number, number]>, cards: string[], withPairs: boolean) {
  const fourRank = rankItems.find(([, count]) => count === 4)?.[0];
  if (!fourRank) {
    return null;
  }

  const attachmentItems = rankItems.filter(([rank]) => rank !== fourRank);
  if (withPairs) {
    if (cards.length !== 8 || attachmentItems.length !== 2 || !attachmentItems.every(([, count]) => count === 2)) {
      return null;
    }
    return buildGroup("four_with_two_pairs", valueToRank(fourRank), 8, cards);
  }

  const attachmentCardCount = attachmentItems.reduce((sum, [, count]) => sum + count, 0);
  if (cards.length !== 6 || attachmentCardCount !== 2 || attachmentItems.some(([, count]) => count > 1)) {
    return null;
  }
  return buildGroup("four_with_two_singles", valueToRank(fourRank), 6, cards);
}

function countByRank(cards: ParsedCard[]) {
  const result = new Map<number, number>();
  for (const card of cards) {
    result.set(card.rankValue, (result.get(card.rankValue) ?? 0) + 1);
  }
  return result;
}

function groupCardsByRank(cards: ParsedCard[]) {
  const result = new Map<number, ParsedCard[]>();
  for (const card of cards) {
    const group = result.get(card.rankValue) ?? [];
    group.push(card);
    result.set(card.rankValue, group);
  }
  return result;
}

function hasRank(cards: ParsedCard[], rank: string) {
  return cards.some((card) => card.rank === rank);
}

function isConsecutive(ranks: number[]) {
  for (let index = 1; index < ranks.length; index++) {
    if (ranks[index] !== ranks[index - 1] + 1) {
      return false;
    }
  }
  return ranks.length > 0;
}

function buildGroup(type: string, rank: string, length: number, cards: string[]): ClientCardGroup {
  return {
    type,
    rank,
    rankValue: rankValues[rank] ?? 0,
    length,
    cards,
  };
}

function valueToRank(value: number) {
  return Object.entries(rankValues).find(([, rankValue]) => rankValue === value)?.[0] ?? "";
}

function dedupeGroups(groups: ClientCardGroup[]) {
  const seen = new Set<string>();
  return groups.filter((group) => {
    const key = `${group.type}:${group.rank}:${group.length}:${group.cards.join(",")}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
