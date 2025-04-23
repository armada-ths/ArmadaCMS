import axiosInstance from "../context/axiosInstance";

export const customFetchAxios = async (
  paramUrl: string,
  params?: Record<string, unknown>,
) => {
  try {
    const response = await axiosInstance({
      url: `/${paramUrl}`,
      method: params ? "POST" : "GET",
      data: params,
    });
    return response.data;
  } catch (error) {
    console.error("API error:", error);
    throw error;
  }
};
