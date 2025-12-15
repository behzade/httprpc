import type {
  CreateParams,
  DataProvider,
  DeleteManyParams,
  DeleteParams,
  GetListParams,
  GetManyParams,
  GetManyReferenceParams,
  GetOneParams,
  Identifier,
  UpdateManyParams,
  UpdateParams,
} from "ra-core";
import { env } from "@/env";
import { Client } from "../../lib/api";
import type {
  CreateProductRequest,
  ListProductsRequest,
  Product,
  UpdateProductRequest,
} from "../../lib/api/api";

const defaultBaseUrl =
  typeof window !== "undefined" && window.location?.origin
    ? window.location.origin
    : "http://localhost:18080";

const client = new Client({
  baseUrl: env.VITE_API_BASE_URL ?? defaultBaseUrl,
});

const ensureProducts = (resource: string) => {
  if (resource !== "products") {
    throw new Error(`Unsupported resource "${resource}"`);
  }
};

const normalizeId = (id: Identifier) => String(id);

const normalizeSortDirection = (order: string | undefined) =>
  order?.toUpperCase() === "DESC" ? "DESC" : "ASC";

const extractQuery = (filter: any): string => {
  if (filter && typeof filter.q === "string") {
    return filter.q;
  }
  if (filter && typeof filter.query === "string") {
    return filter.query;
  }
  return "";
};

const mapListParams = (params: Partial<GetListParams>): ListProductsRequest => {
  const { page = 1, perPage = 25 } = params.pagination ?? {};
  const sortField = params.sort?.field ?? "name";
  const sortDirection = normalizeSortDirection(params.sort?.order ?? "ASC");
  const query = extractQuery(params.filter);

  return {
    page,
    per_page: perPage,
    sort_field: sortField,
    sort_direction: sortDirection,
    query,
  };
};

const mapProductPayload = (
  id: Identifier | undefined,
  data: Partial<Product>,
): UpdateProductRequest => ({
  id: normalizeId(id ?? data.id ?? ""),
  name: data.name ?? "",
  description: data.description ?? "",
  price: Number.isFinite(Number(data.price)) ? Number(data.price) : 0,
  stock: Number.isFinite(Number(data.stock)) ? Number(data.stock) : 0,
});

const mapCreatePayload = (data: Partial<Product>): CreateProductRequest => ({
  name: data.name ?? "",
  description: data.description ?? "",
  price: Number.isFinite(Number(data.price)) ? Number(data.price) : 0,
  stock: Number.isFinite(Number(data.stock)) ? Number(data.stock) : 0,
});

export const dataProvider: DataProvider = {
  async getList(resource: string, params: GetListParams) {
    ensureProducts(resource);
    const req = mapListParams(params);
    const res = await client.api.get_api_products_list(req);
    return { data: res.items as any, total: res.total };
  },

  async getOne(resource: string, params: GetOneParams) {
    ensureProducts(resource);
    const res = await client.api.get_api_products_get({
      id: normalizeId(params.id),
    });
    return { data: res as any };
  },

  async getMany(resource: string, params: GetManyParams) {
    ensureProducts(resource);
    if (!params.ids?.length) {
      return { data: [] };
    }
    const wanted = params.ids.map((id) => normalizeId(id));
    const results = await Promise.all(
      wanted.map((id) => client.api.get_api_products_get({ id })),
    );
    return {
      data: results as any,
    };
  },

  async getManyReference(resource: string, params: GetManyReferenceParams) {
    ensureProducts(resource);
    const req = mapListParams({
      filter: params.filter,
      pagination: params.pagination,
      sort: params.sort,
    });
    const res = await client.api.get_api_products_list(req);
    return { data: res.items as any, total: res.total };
  },

  async create(resource: string, params: CreateParams) {
    ensureProducts(resource);
    const payload = mapCreatePayload(params.data ?? {});
    const data = await client.api.post_api_products(payload);
    return { data: data as any };
  },

  async update(resource: string, params: UpdateParams) {
    ensureProducts(resource);
    const payload = mapProductPayload(params.id, params.data ?? {});
    const data = await client.api.put_api_products(payload);
    return { data: data as any };
  },

  async updateMany(resource: string, params: UpdateManyParams) {
    ensureProducts(resource);
    const updated = await Promise.all(
      params.ids.map((id) =>
        client.api.put_api_products(mapProductPayload(id, params.data ?? {})),
      ),
    );
    return { data: updated.map((item) => item.id) };
  },

  async delete(resource: string, params: DeleteParams) {
    ensureProducts(resource);
    await client.api.delete_api_products({ id: normalizeId(params.id) });
    const previous =
      (params.previousData as Product | undefined) ??
      ({ id: normalizeId(params.id) } as Product);
    return { data: previous as any };
  },

  async deleteMany(resource: string, params: DeleteManyParams) {
    ensureProducts(resource);
    await Promise.all(
      params.ids.map((id) =>
        client.api.delete_api_products({ id: normalizeId(id) }),
      ),
    );
    return { data: params.ids };
  },
};
