import { z } from "zod";

export const loginSchema = z.object({
  email: z.string().trim().email("Wpisz poprawny adres e-mail."),
  password: z.string().min(1, "Wpisz hasło."),
});
