const globalApi = () => {
  let globalApi: string;
  if (import.meta.env.PROD === false) {
    globalApi = "http://localhost:8080/api/v1";
  } else {
    globalApi = `${window.location.origin}/api/v1`;
  }
  return globalApi;
};

export default globalApi;
