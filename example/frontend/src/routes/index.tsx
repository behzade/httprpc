import { createFileRoute } from "@tanstack/react-router";
import { Resource } from "ra-core";
import { Admin } from "@/components/admin";
import { dataProvider } from "@/lib/data-provider";
import {
  ProductCreate,
  ProductEdit,
  ProductList,
} from "@/components/products";

export const Route = createFileRoute("/")({
  component: App,
});

function App() {
  return (
    <Admin dataProvider={dataProvider} title="Products Admin">
      <Resource
        name="products"
        list={ProductList}
        create={ProductCreate}
        edit={ProductEdit}
        recordRepresentation="name"
      />
    </Admin>
  );
}
