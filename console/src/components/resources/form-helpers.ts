import { z } from "zod";

export interface FormValidationResult<T> {
  isValid: boolean;
  errors: Record<string, string>;
  data?: T;
}

export function validateResourceForm<T extends z.ZodRawShape>(
  schema: z.ZodObject<T>,
  values: unknown
): FormValidationResult<z.infer<z.ZodObject<T>>> {
  const result = schema.safeParse(values);
  if (result.success) {
    return {
      isValid: true,
      errors: {},
      data: result.data,
    };
  }

  const errors: Record<string, string> = {};
  for (const issue of result.error.issues) {
    const field = issue.path.join(".") || "form";
    if (!errors[field]) {
      errors[field] = issue.message;
    }
  }

  return {
    isValid: false,
    errors,
  };
}
