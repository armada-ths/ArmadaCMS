import {
  List,
  Datagrid,
  TextField,
  EditButton,
  ListProps,
  DeleteWithConfirmButton,
  ArrayField,
  SingleFieldList,
  ChipField,
} from "react-admin";

export const ExhibitorList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="type" />
      <TextField source="tier" />
      <TextField source="fairLocation" label="Fair location" />
      <ArrayField source="industries">
        <SingleFieldList>
          <ChipField source="name" />
        </SingleFieldList>
      </ArrayField>
      <ArrayField source="programs">
        <SingleFieldList>
          <ChipField source="name" />
        </SingleFieldList>
      </ArrayField>
      <EditButton />
      <DeleteWithConfirmButton
        confirmTitle="Are you sure?"
        confirmContent="This is PERMANENT"
      />
    </Datagrid>
  </List>
);
