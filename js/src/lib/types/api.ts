import type { components } from "@/lib/api/schema.gen";

type Schemas = components["schemas"];

export type User = Schemas["User"];

export type Summary = Schemas["Summary"];

export type CurrencyBalance = Schemas["CurrencyBalance"];

export type CurrencyCashFlow = Schemas["CurrencyCashFlow"];

export type Connection = Schemas["Connection"];

export type ConnectionStatus = Connection["status"];

export type SyncSummary = Schemas["SyncSummary"];

export type Account = Schemas["Account"];

export type Transaction = Schemas["Transaction"];

export type Notification = Schemas["Notification"];

export type ConsentRequest = Schemas["ConsentRequest"];

export type OAuthGrant = Schemas["OAuthGrant"];

export type OAuthScope = ConsentRequest["scopes"][number];
