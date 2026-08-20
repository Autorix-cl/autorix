/**
 * Schema for the honest health aggregator (P1-S4-T1). Boundary-validated
 * like every other engine response (P1-S3-T4): a shape drift here should
 * surface as a precise validation error, not a silently-wrong dashboard.
 */
import { z } from "zod";

export const engineStatusSchema = z.enum(["healthy", "degraded", "unreachable", "unknown"]);
export type EngineStatus = z.infer<typeof engineStatusSchema>;

export const serviceHealthSchema = z.object({
  key: z.string(),
  name: z.string(),
  port: z.number(),
  protocol: z.string(),
  status: engineStatusSchema,
  latencyMs: z.number().nullable(),
});
export type ServiceHealth = z.infer<typeof serviceHealthSchema>;

export const healthResponseSchema = z.object({
  checkedAt: z.string(),
  services: z.array(serviceHealthSchema),
});
export type HealthResponse = z.infer<typeof healthResponseSchema>;
