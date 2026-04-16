import { List, Datagrid, TextField, EditButton, ListProps } from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const HighlightCardList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="title" label="Title" />
      <TextField source="subtitle" label="Subtitle" />
      <TextField source="brand" label="Brand" />
      <TextField source="linkUrl" label="Link URL" />
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
