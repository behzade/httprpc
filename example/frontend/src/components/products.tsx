import {
  Create,
  DataTable,
  Edit,
  EditButton,
  List,
  NumberInput,
  SimpleForm,
  TextInput,
} from "@/components/admin";
import type { Product } from "../../lib/api/api";

export const ProductList = () => (
  <List<Product> sort={{ field: "updated_at", order: "DESC" }}>
    <DataTable<Product>>
      <DataTable.Col source="id" label="ID" disableSort />
      <DataTable.Col source="name" label="Name" />
      <DataTable.Col
        source="description"
        label="Description"
        conditionalClassName={(record) =>
          record.description ? undefined : "text-muted-foreground"
        }
      />
      <DataTable.NumberCol
        source="price"
        label="Price"
        options={{ style: "currency", currency: "USD" }}
      />
      <DataTable.NumberCol source="stock" label="Stock" />
      <DataTable.Col>
        <EditButton />
      </DataTable.Col>
    </DataTable>
  </List>
);

export const ProductCreate = () => (
  <Create>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="description" multiline rows={3} />
      <NumberInput source="price" min={0} step={0.01} />
      <NumberInput source="stock" min={0} step={1} />
    </SimpleForm>
  </Create>
);

export const ProductEdit = () => (
  <Edit>
    <SimpleForm>
      <TextInput source="name" />
      <TextInput source="description" multiline rows={3} />
      <NumberInput source="price" min={0} step={0.01} />
      <NumberInput source="stock" min={0} step={1} />
    </SimpleForm>
  </Edit>
);
