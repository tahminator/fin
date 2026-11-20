import createFetchClient from "openapi-fetch";
import createClient from "openapi-react-query";

import type { paths } from "@/lib/api/schema.gen";

const fetchClient = createFetchClient<paths>({ credentials: "same-origin" });

export const $api = createClient(fetchClient);
