import { httpClient } from "../../dataProvider";
import globalApi from "../../context/globalApi";

export interface LoungeCategory {
  id: number;
  name: string;
  nameSv?: string | null;
  sortOrder: number;
}
export interface LoungeItem {
  id: number;
  categoryId: number;
  name: string;
  soldOut: boolean;
  sortOrder: number;
}
export interface Catalogue {
  revision: number;
  categories: LoungeCategory[];
  items: LoungeItem[];
}
export async function loungeRequest<T>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await httpClient(`${globalApi()}/${path}`, {
    method,
    headers: new Headers({ "Content-Type": "application/json" }),
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  return response.json as T;
}

export function moveCatalogue(
  catalogue: Catalogue,
  active: string,
  over: string,
): Catalogue {
  const next = {
    ...catalogue,
    categories: [...catalogue.categories],
    items: catalogue.items.map((item) => ({ ...item })),
  };
  const categoryOf = (key: string) =>
    key.startsWith("category-")
      ? Number(key.slice(9))
      : next.items.find((item) => item.id === Number(key.slice(5)))?.categoryId;
  if (active.startsWith("category-")) {
    const from = next.categories.findIndex(
      (category) => category.id === categoryOf(active),
    );
    const to = next.categories.findIndex(
      (category) => category.id === categoryOf(over),
    );
    if (from < 0 || to < 0) return catalogue;
    const [category] = next.categories.splice(from, 1);
    next.categories.splice(to, 0, category);
    next.categories = next.categories.map((category, sortOrder) => ({
      ...category,
      sortOrder,
    }));
    return next;
  }
  const item = next.items.find((item) => item.id === Number(active.slice(5)));
  const destination = categoryOf(over);
  if (!item || destination == null) return catalogue;
  const source = item.categoryId;
  const sameCategory = source === destination;
  const group = next.items
    .filter((entry) => entry.categoryId === destination)
    .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id);
  const from = group.findIndex((entry) => entry.id === item.id);
  const target = over.startsWith("item-")
    ? group.findIndex((entry) => entry.id === Number(over.slice(5)))
    : group.length;
  const remaining = group.filter((entry) => entry.id !== item.id);
  const index = Math.min(
    target < 0 ? remaining.length : target,
    remaining.length,
  );
  if (sameCategory && from === target) return catalogue;
  remaining.splice(index, 0, item);
  item.categoryId = destination;
  remaining.forEach((entry, position) => {
    entry.sortOrder = position;
  });
  if (!sameCategory)
    next.items
      .filter((entry) => entry.categoryId === source)
      .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
      .forEach((entry, position) => {
        entry.sortOrder = position;
      });
  return next;
}

export function reorderPayload(catalogue: Catalogue) {
  return {
    revision: catalogue.revision,
    categories: catalogue.categories.map((category) => ({
      id: category.id,
      itemIds: catalogue.items
        .filter((item) => item.categoryId === category.id)
        .sort((a, b) => a.sortOrder - b.sortOrder || a.id - b.id)
        .map((item) => item.id),
    })),
  };
}
