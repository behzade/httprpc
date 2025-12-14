import { createFileRoute } from "@tanstack/react-router";
import { Admin } from "@/components/admin";

export const Route = createFileRoute("/")({
    component: App,
});

function App() {
    return <Admin></Admin>;
}
