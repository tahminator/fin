export function groupBy<Item, Key>(items: Item[], keyOf: (item: Item) => Key): Map<Key, Item[]> {
  const groups = new Map<Key, Item[]>();

  for (const item of items) {
    const key = keyOf(item);
    const group = groups.get(key);

    if (group === undefined) {
      groups.set(key, [item]);
    } else {
      group.push(item);
    }
  }

  return groups;
}
