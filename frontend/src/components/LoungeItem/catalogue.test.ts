import { describe, expect, it } from "vitest";
import { Catalogue, moveCatalogue, reorderPayload } from "./catalogue";

const catalogue: Catalogue = {
  revision: 3,
  categories: [
    { id: 1, name: "Drinks", sortOrder: 0 },
    { id: 2, name: "Snacks", sortOrder: 1 },
  ],
  items: [
    { id: 1, name: "Coffee", categoryId: 1, soldOut: false, sortOrder: 0 },
    { id: 2, name: "Tea", categoryId: 1, soldOut: false, sortOrder: 1 },
    { id: 3, name: "Bun", categoryId: 2, soldOut: false, sortOrder: 0 },
  ],
};
describe("catalogue drag ordering", () => {
  it("reorders categories without changing item membership", () => {
    const next = moveCatalogue(catalogue, "category-2", "category-1");
    expect(next.categories.map((category) => category.id)).toEqual([2, 1]);
    expect(next.items).toEqual(catalogue.items);
  });
  it("reorders items within a category without mutating its input", () => {
    const next = moveCatalogue(catalogue, "item-1", "item-2");
    expect(reorderPayload(next).categories[0].itemIds).toEqual([2, 1]);
    expect(catalogue.items[0].sortOrder).toBe(0);
  });
  it("moves items into another category and compacts both positions", () => {
    const next = moveCatalogue(catalogue, "item-1", "item-3");
    expect(reorderPayload(next)).toEqual({
      revision: 3,
      categories: [
        { id: 1, itemIds: [2] },
        { id: 2, itemIds: [1, 3] },
      ],
    });
    expect(next.items.find((item) => item.id === 2)?.sortOrder).toBe(0);
  });
  it("supports dropping into an empty category", () => {
    const empty = {
      ...catalogue,
      items: catalogue.items.filter((item) => item.categoryId !== 2),
    };
    expect(
      reorderPayload(moveCatalogue(empty, "item-1", "category-2")).categories[1]
        .itemIds,
    ).toEqual([1]);
  });
});
