import createClient from "openapi-fetch";
import type { paths } from "./schema";
import { authMiddleware, unauthorizedMiddleware } from "./auth";

const client = createClient<paths>({
  baseUrl: "/api/v1",
});

client.use(authMiddleware);
client.use(unauthorizedMiddleware);

export default client;
