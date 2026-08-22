import { useState } from "react";
import { Button, Stack, TextField, Typography } from "@mui/material";
import { DateInput, SimpleForm, TextInput } from "react-admin";
import { useController } from "react-hook-form";

const splitFairDays = (value: unknown): string[] => {
  if (typeof value !== "string") {
    return [];
  }

  return value
    .split(",")
    .map((day) => day.trim())
    .filter(Boolean);
};

const joinFairDays = (values: string[]): string =>
  values
    .map((value) => value.trim())
    .filter(Boolean)
    .join(",");

const FairDaysInput = () => {
  const {
    field,
    fieldState: { error, invalid },
  } = useController({
    name: "fairDays",
    defaultValue: "",
  });
  const [draftValues, setDraftValues] = useState<string[] | null>(null);
  const values = draftValues ?? splitFairDays(field.value);

  const syncFieldValue = (nextValues: string[]) => {
    field.onChange(joinFairDays(nextValues));
  };

  const handleDateChange = (index: number, nextValue: string) => {
    const nextValues = [...values];
    nextValues[index] = nextValue;
    setDraftValues(nextValues);
    syncFieldValue(nextValues);
  };

  const handleAddDate = () => {
    setDraftValues([...values, ""]);
  };

  const handleRemoveDate = (index: number) => {
    const nextValues = values.filter(
      (_, currentIndex) => currentIndex !== index,
    );
    setDraftValues(nextValues);
    syncFieldValue(nextValues);
  };

  return (
    <Stack spacing={1.5} sx={{ mb: 1.5, mt: 0.5 }}>
      <Typography variant="body2">Fair Days</Typography>
      {values.length === 0 && (
        <Typography variant="body2" color="text.secondary">
          Add one or more fair days using the button below.
        </Typography>
      )}
      {values.map((value, index) => (
        <Stack
          key={`${index}-${value || "empty"}`}
          direction="row"
          spacing={1.5}
        >
          <TextField
            label={`Fair day ${index + 1}`}
            type="date"
            value={value}
            onChange={(event) => handleDateChange(index, event.target.value)}
            onBlur={field.onBlur}
            error={invalid}
            helperText={index === 0 ? error?.message : undefined}
            slotProps={{
              inputLabel: {
                shrink: true,
              },
            }}
            fullWidth
          />
          <Button
            variant="outlined"
            color="error"
            onClick={() => handleRemoveDate(index)}
          >
            Remove
          </Button>
        </Stack>
      ))}
      <Button variant="outlined" onClick={handleAddDate}>
        Add fair day
      </Button>
    </Stack>
  );
};

export const FairDateForm = () => (
  <SimpleForm>
    <TextInput
      source="description"
      label="Description"
      helperText='e.g. "THS Armada 2026"'
    />

    <FairDaysInput />

    <DateInput
      source="irStart"
      label="IR Start"
      helperText="Initial Registration start date"
    />
    <DateInput
      source="irEnd"
      label="IR End"
      helperText="Initial Registration end date"
    />
    <DateInput
      source="irAcceptance"
      label="IR Acceptance"
      helperText="Acceptance notification date"
    />
    <DateInput
      source="frStart"
      label="FR Start"
      helperText="Final Registration start date"
    />
    <DateInput
      source="frEnd"
      label="FR End"
      helperText="Final Registration end date"
    />
    <DateInput
      source="eventsStart"
      label="Events Start"
      helperText="Events week start date"
    />
  </SimpleForm>
);
