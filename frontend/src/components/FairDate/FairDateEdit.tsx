import { Edit, EditProps, SimpleForm, TextInput } from "react-admin";

export const FairDateEdit = (props: EditProps) => (
  <Edit {...props}>
    <SimpleForm>
      <TextInput
        source="description"
        label="Description"
        helperText='e.g. "THS Armada 2026"'
      />
      <TextInput
        source="fairDays"
        label="Fair Days"
        helperText="Comma-separated dates, e.g. 2026-11-17,2026-11-18"
      />
      <TextInput
        source="ticketEnd"
        label="Ticket End Date"
        helperText="Leave empty if no ticket deadline"
      />
      <TextInput
        source="irStart"
        label="IR Start"
        helperText="Initial Registration start date (YYYY-MM-DD)"
      />
      <TextInput
        source="irEnd"
        label="IR End"
        helperText="Initial Registration end date"
      />
      <TextInput
        source="irAcceptance"
        label="IR Acceptance"
        helperText="Acceptance notification date"
      />
      <TextInput
        source="frStart"
        label="FR Start"
        helperText="Final Registration start date"
      />
      <TextInput
        source="frEnd"
        label="FR End"
        helperText="Final Registration end date"
      />
      <TextInput
        source="eventsStart"
        label="Events Start"
        helperText="Events week start date"
      />
      <TextInput
        source="eventsEnd"
        label="Events End"
        helperText="Events end date (leave empty if N/A)"
      />
    </SimpleForm>
  </Edit>
);
