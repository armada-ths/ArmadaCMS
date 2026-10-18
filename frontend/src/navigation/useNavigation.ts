import { useCanAccessResources } from "react-admin";
import { filterNavigation, NAVIGATION, RESOURCE_NAMES } from "./catalog";

export const useNavigation = () => {
  const { canAccess, isPending, error } = useCanAccessResources({
    action: "list",
    resources: RESOURCE_NAMES,
  });
  return {
    nodes: filterNavigation(
      NAVIGATION,
      !isPending && !error ? (canAccess ?? {}) : {},
    ),
    isPending,
    error,
  };
};
