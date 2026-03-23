import {
  DateField,
  Datagrid,
  EditButton,
  List,
  ListProps,
  TextField,
} from "react-admin";
import { PermissionDeleteButton } from "../shared/PermissionDeleteButton";

export const RecruitmentPeriodList = (props: ListProps) => (
  <List {...props}>
    <Datagrid>
      <TextField source="id" />
      <TextField source="name" />
      <TextField source="link" />
      <DateField source="startDate" showTime={false} />
      <DateField source="endDate" showTime={false} />
      <EditButton />
      <PermissionDeleteButton />
    </Datagrid>
  </List>
);
