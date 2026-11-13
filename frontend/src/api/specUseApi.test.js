import { fetchProducts, validateBuild } from "./specUseApi";

describe("SpecUse API client", () => {
  beforeEach(() => {
    global.fetch = vi.fn();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("loads and paginates active products", async () => {
    const firstPageItems = Array.from({ length: 100 }, (_, index) => ({
      id: `product-${index + 1}`,
      name: `Product ${index + 1}`,
      category: "other",
      brand: "Example",
      price_thb: 100,
      is_active: true,
    }));
    global.fetch
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            items: firstPageItems,
            page: 1,
            page_size: 100,
            total: 101,
          },
        }),
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          data: {
            items: [{
              id: "product-101",
              name: "Product 101",
              category: "other",
              brand: "Example",
              price_thb: 100,
              is_active: true,
            }],
            page: 2,
            page_size: 100,
            total: 101,
          },
        }),
      });

    await expect(fetchProducts()).resolves.toHaveLength(101);
    expect(global.fetch).toHaveBeenCalledTimes(2);
  });

  it("posts only the supplied build data and returns the data envelope", async () => {
    const build = { items: [], ram_items: [] };
    global.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { compatible: true, total_price_thb: 0 } }),
    });

    await expect(validateBuild(build)).resolves.toEqual({
      compatible: true,
      total_price_thb: 0,
    });
    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1",
      expect.objectContaining({
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(build),
      }),
    );
  });

  it("passes an optional abort signal to build validation", async () => {
    const controller = new AbortController();
    global.fetch.mockResolvedValue({
      ok: true,
      json: async () => ({ data: { compatible: true, total_price_thb: 0 } }),
    });

    await validateBuild({ items: [], ram_items: [] }, { signal: controller.signal });

    expect(global.fetch).toHaveBeenCalledWith(
      "http://127.0.0.1:8080/api/v1/builds/validate?validation_mode=gpu_storage_v1",
      expect.objectContaining({ signal: controller.signal }),
    );
  });

  it("surfaces API errors to the caller", async () => {
    global.fetch.mockResolvedValue({
      ok: false,
      status: 400,
      json: async () => ({ error: { code: "VALIDATION_ERROR", message: "ข้อมูลไม่ถูกต้อง" } }),
    });

    await expect(validateBuild({ items: [], ram_items: [] })).rejects.toThrow("ข้อมูลไม่ถูกต้อง");
  });
});
