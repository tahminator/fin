export const pageSize = 25;

export function nextPage(last: { page: number; hasNext: boolean }): number | undefined {
  return last.hasNext ? last.page + 1 : undefined;
}

export function flattenPages<Item>(data: { pages: { items: Item[] }[] }): Item[] {
  return data.pages.flatMap((page) => page.items);
}
