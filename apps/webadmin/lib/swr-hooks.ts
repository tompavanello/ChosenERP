import useSWR from "swr";
import useSWRMutation from "swr/mutation";
import { mutate as swrMutate } from "swr";
import {
  type Member,
  type Visitor,
  type Benefactor,
  type Family,
  type Category,
  type Transaction,
  type Balance,
  type Branch,
  type Person,
  type PeopleQuery,
  type Transfer,
  listMembers,
  getMember,
  listFamilies,
  listVisitors,
  listBenefactors,
  listCategories,
  listTransactions,
  getBalance,
  listBranches,
  listPeople,
  listTransfers,
  createMember,
  updateMember,
  createPerson,
  updatePerson,
  createVisitor,
  createBenefactor,
  createFamily,
  createCategory,
  createTransaction,
} from "@/lib/api";

// Fetcher generico para SWR. O caminho e relativo: a API e servida na mesma
// origem do webadmin (ver o comentario no topo de lib/api.ts).
//
// Ressalva conhecida: este fetcher le o token do localStorage direto e nao passa
// pelo api(), entao NAO faz refresh em 401 - o comportamento dele difere do das
// chamadas que usam api() quando o access token expira.
const fetcher = (url: string) => fetch(url, {
  headers: { Authorization: `Bearer ${localStorage.getItem("chosen_token") ?? ""}` },
}).then((r) => {
  if (!r.ok) throw new Error(`HTTP ${r.status}`);
  return r.json();
});

// Invalida o cache SWR usando prefixo. `mutate("people")` invalida a chave
// exata "people" e todas as que comecam com "people:" (ex.: "people:{...}").
export function mutate(key: string) {
  return swrMutate((k) => typeof k === "string" && (k === key || k.startsWith(`${key}:`)));
}

// ---- Hooks de leitura com cache ----

export function useMembers() {
  return useSWR<{ members: Member[] }>(
    "members",
    async () => listMembers(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

export function useMember(id: string | null) {
  return useSWR<Member>(
    id ? `member:${id}` : null,
    id ? () => getMember(id) : null,
    { revalidateOnFocus: false }
  );
}

export function usePeople(query: PeopleQuery = {}) {
  const key = `people:${JSON.stringify(query)}`;
  return useSWR<{ people: Person[]; total: number }>(
    key,
    () => listPeople(query),
    { revalidateOnFocus: false, dedupingInterval: 30000 }
  );
}

export function useFamilies() {
  return useSWR<{ families: Family[] }>(
    "families",
    () => listFamilies(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

export function useVisitors() {
  return useSWR<{ visitors: Visitor[] }>(
    "visitors",
    () => listVisitors(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

export function useBenefactors() {
  return useSWR<{ benefactors: Benefactor[] }>(
    "benefactors",
    () => listBenefactors(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

export function useCategories() {
  return useSWR<{ categories: Category[] }>(
    "categories",
    () => listCategories(),
    { revalidateOnFocus: false, dedupingInterval: 60000 }
  );
}

export function useTransactions(type = "") {
  const key = type ? `transactions:${type}` : "transactions";
  return useSWR<{ transactions: Transaction[] }>(
    key,
    () => listTransactions(type),
    { revalidateOnFocus: false, dedupingInterval: 30000 }
  );
}

export function useBalance() {
  return useSWR<Balance>(
    "balance",
    () => getBalance(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

export function useBranches() {
  return useSWR<{ branches: Branch[] }>(
    "branches",
    () => listBranches(),
    { revalidateOnFocus: false, dedupingInterval: 60000 }
  );
}

export function useTransfers() {
  return useSWR<{ transfers: Transfer[] }>(
    "transfers",
    () => listTransfers(),
    { revalidateOnFocus: true, dedupingInterval: 30000 }
  );
}

// ---- Hooks de mutacao com invalidacao ----

export function useCreateMember() {
  return useSWRMutation(
    "member:create",
    async (_: string, { arg }: { arg: Record<string, unknown> }) => createMember(arg),
    {
      onSuccess: () => {
        mutate("members");
        mutate("people");
      },
    }
  );
}

export function useUpdateMember() {
  return useSWRMutation(
    "member:update",
    async (_: string, { arg }: { arg: { id: string; data: Record<string, unknown> } }) => updateMember(arg.id, arg.data),
    {
      onSuccess: () => {
        mutate("members");
        mutate("people");
      },
    }
  );
}

export function useCreateVisitor() {
  return useSWRMutation(
    "visitor:create",
    async (_: string, { arg }: { arg: Record<string, string> }) => createVisitor(arg),
    {
      onSuccess: () => {
        mutate("visitors");
        mutate("people");
      },
    }
  );
}

export function useCreateBenefactor() {
  return useSWRMutation(
    "benefactor:create",
    async (_: string, { arg }: { arg: Record<string, string> }) => createBenefactor(arg),
    {
      onSuccess: () => {
        mutate("benefactors");
        mutate("people");
      },
    }
  );
}

export function useCreatePerson() {
  return useSWRMutation(
    "person:create",
    async (_: string, { arg }: { arg: { type: Person["type"]; data: Record<string, unknown> } }) =>
      createPerson(arg.type, arg.data),
    {
      onSuccess: () => mutate("people"),
    }
  );
}

export function useUpdatePerson() {
  return useSWRMutation(
    "person:update",
    async (_: string, { arg }: { arg: { type: Person["type"]; id: string; data: Record<string, unknown> } }) =>
      updatePerson(arg.type, arg.id, arg.data),
    {
      onSuccess: () => {
        mutate("people");
      },
    }
  );
}

export function useCreateFamily() {
  return useSWRMutation(
    "family:create",
    async (_: string, { arg }: { arg: string }) => createFamily(arg),
    {
      onSuccess: () => mutate("families"),
    }
  );
}

export function useCreateCategory() {
  return useSWRMutation(
    "category:create",
    async (_: string, { arg }: { arg: Record<string, unknown> }) => createCategory(arg),
    {
      onSuccess: () => mutate("categories"),
    }
  );
}

export function useCreateTransaction() {
  return useSWRMutation(
    "transaction:create",
    async (_: string, { arg }: { arg: Record<string, unknown> }) => createTransaction(arg),
    {
      onSuccess: () => mutate("transactions"),
    }
  );
}
