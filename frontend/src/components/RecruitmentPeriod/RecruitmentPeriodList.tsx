import {
  DateField,
  Datagrid,
  DeleteButton,
  EditButton,
  List,
  ListProps,
  TextField,
} from "react-admin";

export const RecruitmentPeriodList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="link" />
      <DateField source="startDate" showTime={false} />
      <DateField source="endDate" showTime={false} />
      <EditButton />
      <DeleteButton />
    </Datagrid>
  </List>
);
