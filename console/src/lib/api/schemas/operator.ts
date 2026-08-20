import { z } from "zod";

export const operatorSchema = z.object({
  id: z.string().uuid(),
  email: z.string().email(),
  name: z.string(),
  role: z.enum(["owner", "admin", "operator", "auditor"]),
  is_local: z.boolean(),
  is_active: z.boolean(),
  created_at: z.string().optional(),
  updated_at: z.string().optional(),
});

export const operatorsListSchema = z.array(operatorSchema);

export type OperatorDTO = z.infer<typeof operatorSchema>;
