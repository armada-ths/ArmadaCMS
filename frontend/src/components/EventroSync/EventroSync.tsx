import { useNotify, useAuthenticated } from "react-admin";
import globalApi from "../../context/globalApi";

export const EventroSync = () => {
    const notify = useNotify();
    useAuthenticated();

    const endpoints = [
        { name: "Exhibitors", path: "/eventroexhibitors" },
        { name: "Events", path: "/eventroevents" },
        { name: "Members", path: "/eventromembers" },
        { name: "Recruitments", path: "/eventrorecruitments" },
    ] as const;

    const handleSync = async (name: string, path: string) => {
        try {
            const res = await fetch(`${globalApi()}${path}`, {
                method: "GET",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: `Bearer ${localStorage.getItem("accessToken")}`,
                },
            });

            const message = await res.text();

            if (!res.ok) {
                notify(`❌ ${name}: ${res.status} ${message}`, { type: "error" });
                return;
            }

            notify(`✅ ${name}: ${message}`, { type: "info" });
        } catch (err) {
            notify(`🚨 ${name}: Unexpected error: ${err}`, { type: "error" });
        }
    };

    return (
        <div style={{ padding: "12px" }}>
            <div
                style={{
                    display: "flex",
                    flexWrap: "wrap",
                    gap: "12px",
                    margin: "8px 0 12px 0",
                }}
            >
                {endpoints.map((endpoint) => (
                    <button
                        key={endpoint.path}
                        onClick={() => handleSync(endpoint.name, endpoint.path)}
                        className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition"
                        style={{ margin: "6px" }}
                    >
                        Sync {endpoint.name}
                    </button>
                ))}
            </div>
            <p className="text-sm text-gray-500">
                Press the buttons above to synchronize data from the Eventro API.
            </p>
            <p className="text-xs text-gray-400" style={{ marginTop: "6px" }}>
                Note: Recruitment and profile sync are non-destructive and only fill
                missing data.
            </p>
        </div>
    );
};
