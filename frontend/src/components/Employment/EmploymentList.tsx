import {
  List,
  Datagrid,
  TextField,
  ReferenceField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
} from "react-admin";

export const EmploymentList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" />
      <ReferenceField source="exhibitor_id" reference="exhibitors">
        <TextField source="name" />
      </ReferenceField>
      <EditButton />
      <DeleteWithConfirmButton
        confirmTitle="Are you sure?"
        confirmContent="This is PERMANENT"
      />
    </Datagrid>
  </List>
);
