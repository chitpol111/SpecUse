import { getOwnedQuantitiesAfterSelection } from "./buildSelection";

describe("getOwnedQuantitiesAfterSelection", () => {
  const quantities = { cpu: 1, motherboard: 1, ram: 2, gpu: 1, storage: 1, psu: 1, case: 1 };

  it.each(["cpu", "motherboard", "ram", "gpu", "storage", "psu", "case"])(
    "resets the owned quantity when the selected %s changes",
    (category) => {
      expect(
        getOwnedQuantitiesAfterSelection(
          quantities,
          category,
          "previous-item",
          "next-item",
        ),
      ).toEqual({ ...quantities, [category]: 0 });
    },
  );

  it("preserves quantities when the same item remains selected", () => {
    expect(
      getOwnedQuantitiesAfterSelection(quantities, "cpu", "same-item", "same-item"),
    ).toBe(quantities);
  });
});
