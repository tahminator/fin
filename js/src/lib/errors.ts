import { z } from "zod";

const problemSchema = z.object({
  status: z.number().optional(),
  title: z.string().optional(),
  detail: z.string().optional(),
});

export function problemStatus(error: unknown): number | null {
  const problem = problemSchema.safeParse(error);

  return problem.data?.status ?? null;
}

export function errorMessage(error: unknown): string {
  const problem = problemSchema.safeParse(error);
  const message = problem.data?.detail ?? problem.data?.title;

  if (message !== undefined && message.length > 0) {
    return message;
  }

  if (error instanceof Error) {
    return error.message;
  }

  return "Something went wrong.";
}
