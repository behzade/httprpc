import { Outlet, createRootRouteWithContext } from "@tanstack/react-router";
import { Devtools } from "../components/devtools";

import type { QueryClient } from '@tanstack/react-query'

interface MyRouterContext {
  queryClient: QueryClient
}

export const Route = createRootRouteWithContext<MyRouterContext>()({
  component: () => (
    <>
      <Outlet />
      <Devtools />
    </>
  ),
})
