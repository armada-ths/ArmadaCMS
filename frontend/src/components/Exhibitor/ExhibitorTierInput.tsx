import { SelectInput } from "react-admin";

const tierChoices = ["Standard", "Bronze", "Silver", "Gold"].map((tier) => ({
  id: tier,
  name: tier,
}));

export const ExhibitorTierInput = () => (
  <SelectInput source="tier" choices={tierChoices} emptyText="Not set" />
);
