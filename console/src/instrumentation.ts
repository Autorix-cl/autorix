// Next.js server startup hook: delegate Node-only networking setup to a
// sibling module so the Edge runtime bundle never imports Node core modules.
export async function register(): Promise<void> {
  if (process.env.NEXT_RUNTIME === "nodejs") {
    const { registerNodeNetworking } = await import("./instrumentation-node");
    await registerNodeNetworking();
  }
}
