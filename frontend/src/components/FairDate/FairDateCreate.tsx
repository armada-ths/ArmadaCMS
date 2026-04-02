import { Create, CreateProps } from "react-admin";
import { FairDateForm } from "./FairDateForm";

export const FairDateCreate = (props: CreateProps) => (
  <Create {...props}>
    <FairDateForm />
  </Create>
);
