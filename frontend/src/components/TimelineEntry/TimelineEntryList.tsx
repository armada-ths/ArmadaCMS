import {
  List,
  Datagrid,
  TextField,
  NumberField,
  DeleteButton,
  EditButton,
  ListProps,
  ReferenceField,
} from "react-admin";

export const TimelineEntryList = (props: ListProps) => {
  return (
    <List {...props} sort={{ field: "timelineOrder", order: "ASC" }}>
      <Datagrid>
        <ReferenceField
          source="eraId"
          reference="timeline-eras"
          label="Era"
          sortable={false}
        >
          <TextField source="title" />
        </ReferenceField>
        <NumberField
          source="sortOrder"
          label="Position in era"
          sortable={false}
        />
        <TextField source="title" label="Event title" sortable={false} />
        <TextField
          source="body"
          label="Body preview"
          sortable={false}
          sx={{
            maxWidth: 400,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        />
        <EditButton />
        <DeleteButton />
      </Datagrid>
    </List>
  );
};
