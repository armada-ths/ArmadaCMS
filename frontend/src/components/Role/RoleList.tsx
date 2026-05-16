import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
  ArrayField,
  SingleFieldList,
  ChipField,
} from "react-admin";

export const RoleList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <ArrayField source="permissions">
        <SingleFieldList linkType={false}>
          <ChipField source="" />
        </SingleFieldList>
      </ArrayField>
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
