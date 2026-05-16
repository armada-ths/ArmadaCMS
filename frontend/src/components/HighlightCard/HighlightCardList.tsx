import {
  List,
  Datagrid,
  TextField,
  DeleteButton,
  EditButton,
  ListProps,
} from "react-admin";

export const HighlightCardList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" label="Title" />
      <TextField source="subtitle" label="Subtitle" />
      <TextField source="brand" label="Brand" />
      <TextField source="linkUrl" label="Link URL" />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
