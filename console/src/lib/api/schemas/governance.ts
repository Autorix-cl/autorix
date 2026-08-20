import { z } from "zod";

export const organisationSchema = z.object({
  id: z.string(),
  name: z.string(),
  slug: z.string(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
  status: z.string().optional(),
  plan: z.string().optional(),
});

export type Organisation = z.infer<typeof organisationSchema>;

export const organisationsListSchema = z.union([
  z.array(organisationSchema),
  z.object({
    data: z.array(organisationSchema),
    next_cursor: z.string().optional(),
    has_more: z.boolean().optional(),
  }),
]);

export const projectSchema = z.object({
  id: z.string(),
  org_id: z.string(),
  name: z.string(),
  slug: z.string(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});

export type Project = z.infer<typeof projectSchema>;

export const projectsListSchema = z.union([
  z.array(projectSchema),
  z.object({
    data: z.array(projectSchema),
    next_cursor: z.string().optional(),
    has_more: z.boolean().optional(),
  }),
]);

export const environmentTierSchema = z.enum(["production", "staging", "development", "sandbox"]);

export const environmentSchema = z.object({
  id: z.string(),
  project_id: z.string().optional(),
  name: z.string(),
  slug: z.string(),
  tier: environmentTierSchema.optional(),
  is_production: z.boolean().optional(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});

export type GovernanceEnvironment = z.infer<typeof environmentSchema>;

export const environmentsListSchema = z.union([
  z.array(environmentSchema),
  z.object({
    data: z.array(environmentSchema),
    next_cursor: z.string().optional(),
    has_more: z.boolean().optional(),
  }),
]);
