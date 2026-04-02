import { Edit, EditProps } from "react-admin";
import { FairDateForm } from "./FairDateForm";

export const FairDateEdit = (props: EditProps) => (
  <Edit {...props}>
    <FairDateForm />
  </Edit>
);
