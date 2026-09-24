"use client";

import {
  PersonForm,
  toPersonForm,
  compactPersonForm,
  type PersonFormProps,
  type PersonFormState,
  type PersonInput,
} from "@/components/people/person-form";

export function MemberForm(props: Omit<PersonFormProps, "entityType">) {
  return <PersonForm {...props} entityType="member" />;
}
export { toPersonForm as toMemberForm, compactPersonForm as compactMemberForm };
export type { PersonFormProps as MemberFormProps, PersonFormState as FormState, PersonInput };
