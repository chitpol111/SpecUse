const OWNED_QUANTITY_CATEGORIES = new Set(["cpu", "motherboard", "ram"]);

export function getOwnedQuantitiesAfterSelection(
  quantities,
  category,
  previousID,
  nextID,
) {
  if (
    previousID === nextID ||
    !OWNED_QUANTITY_CATEGORIES.has(category)
  ) {
    return quantities;
  }

  return { ...quantities, [category]: 0 };
}
