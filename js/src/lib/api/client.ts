import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";

import type { paths } from "@/lib/api/schema.gen";
import { fakeLagFetch } from "@/lib/fake-lag";

const fetchClient = createFetchClient<paths>({
  credentials: "same-origin",
  fetch: fakeLagFetch,
});

export const $api = createClient(fetchClient);
