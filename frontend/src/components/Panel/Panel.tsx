import { useNotify, useAuthenticated } from "react-admin";
import globalApi from "../../context/globalApi";

export const Panel = () => {
  const notify = useNotify();
  useAuthenticated();

  const handleFetch = async () => {
    const endpoints = [
      { name: "Exhibitors", path: "/eventroexhibitors" },
      { name: "Events", path: "/eventroevents" },
      { name: "Members", path: "/eventromembers" },
    ];

    try {
      // Run both in parallel
      const results = await Promise.allSettled(
        endpoints.map((ep) =>
          fetch(`${globalApi()}${ep.path}`, {
            method: "GET",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${localStorage.getItem("accessToken")}`,
            },
          }).then(async (res) => {
            if (!res.ok) {
              const msg = await res.text();
              throw new Error(`${ep.name}: ${res.status} ${msg}`);
            }
            return `${ep.name} ✓`;
          }),
        ),
      );

      // Build a message summarizing results
      const success = results
        .filter((r) => r.status === "fulfilled")
        .map((r) => r.value)
        .join(", ");
      const failed = results
        .filter((r) => r.status === "rejected")
        .map((r) => (r.reason as Error).message)
        .join("\n");

      if (failed) {
        notify(`❌ Some requests failed:\n${failed}`, { type: "error" });
      }
      if (success) {
        notify(`✅ Success: ${success}`, { type: "info" });
      }
    } catch (err) {
      notify(`🚨 Unexpected error: ${err}`, { type: "error" });
    }
  };

  return (
    <div className="p-4 space-y-4">
      <button
        onClick={handleFetch}
        className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition"
      >
        Trigger Eventro Sync
      </button>
      <p className="text-sm text-gray-500">
        This will sync Exhibitors, Events, and Team Members from Eventro.
      </p>
    </div>
  );
};
