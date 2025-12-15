import { Suspense, lazy } from "react";

const LazyDevtools =
  import.meta.env.DEV
    ? lazy(async () => {
        const [{ TanStackDevtools }, { TanStackRouterDevtoolsPanel }, TanStackQueryDevtools] =
          await Promise.all([
            import("@tanstack/react-devtools"),
            import("@tanstack/react-router-devtools"),
            import("../integrations/tanstack-query/devtools"),
          ]);
        const queryPlugin =
          // TanStackQueryDevtools exports a default plugin object
          (TanStackQueryDevtools as { default?: unknown }).default ??
          TanStackQueryDevtools;
        return {
          default: () => (
            <TanStackDevtools
              config={{ position: "bottom-right" }}
              plugins={[
                { name: "Tanstack Router", render: <TanStackRouterDevtoolsPanel /> },
                queryPlugin as any,
              ]}
            />
          ),
        };
      })
    : null;

export function Devtools() {
  if (!LazyDevtools) return null;
  const Component = LazyDevtools;
  return (
    <Suspense fallback={null}>
      <Component />
    </Suspense>
  );
}
