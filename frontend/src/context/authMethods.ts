import { useMutation } from "@tanstack/react-query";
import { customFetchAxios } from "../utils/customFetchAxios";
import { Tokens, UserLogin } from "../models/user";

export const loginApi = async (user: UserLogin): Promise<Tokens> => {
  const tokens = await customFetchAxios("login", { ...user });

  if (!tokens.accessToken || !tokens.refreshToken) {
    throw new Error("Invalid tokens received");
  }

  return tokens;
};
export const useLogin = () => {
  return useMutation<Tokens, Error, UserLogin>({
    mutationFn: loginApi,
    onSuccess: (tokens) => {
      localStorage.setItem("accessToken", tokens.accessToken);
      localStorage.setItem("refreshToken", tokens.refreshToken);
    },
  });
};
