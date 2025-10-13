import { useNotify, useAuthenticated } from "react-admin";
import globalApi from "../../context/globalApi";

export const Panel = () => {
  const notify = useNotify();
  useAuthenticated();

  const handleFetch = async () => {
    try {
      const res = await fetch(`${globalApi()}/eventroexhibitors`, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("auth")}`,
        },
      });

      if (!res.ok) throw new Error(`Status ${res.status}`);
      notify("Request succeeded", { type: "info" });
    } catch (err) {
      notify(`Request failed: ${err}`, { type: "error" });
    }
  };

  return (
    <div className="p-4">
      <button
        onClick={handleFetch}
        className="bg-blue-600 text-white px-4 py-2 rounded"
      >
        Trigger GET
      </button>
    </div>
  );
};
